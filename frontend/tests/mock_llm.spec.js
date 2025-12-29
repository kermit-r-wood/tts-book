import { test, expect } from '@playwright/test';

test.describe('Mock LLM Analysis Flow', () => {
    test('Complete Flow: Upload -> Settings -> Analyze', async ({ page }) => {
        // Set a large viewport
        await page.setViewportSize({ width: 1280, height: 800 });

        page.on('console', msg => console.log(`[Browser Console] ${msg.text()}`));
        page.on('pageerror', err => console.log(`[Browser Error] ${err.message}`));

        // Mock WebSocket
        await page.addInitScript(() => {
            class MockWebSocket {
                constructor(url) {
                    console.log(`[MockWS] Connected to ${url}`);
                    setTimeout(() => this.onopen && this.onopen(), 100);
                }
                close() { }
                send() { }
            }
            window.WebSocket = MockWebSocket;
        });

        // 1. Mock Network Requests using Globs
        await page.route('**/api/config', async route => {
            const method = route.request().method();
            if (method === 'POST') {
                await route.fulfill({ json: { status: 'ok' } });
            } else {
                // GET
                await route.fulfill({
                    json: {
                        llm_chunk_size: 1000,
                        llm_min_interval: 3000,
                        mock_llm: false,
                        voice_dir: ''
                    }
                });
            }
        });

        await page.route('**/api/upload', async route => {
            await route.fulfill({
                json: {
                    message: "Upload successful",
                    chapters: [
                        { id: "ch_mock", title: "Mock Chapter 1", content: "Once upon a time..." }
                    ],
                    bookPath: "/tmp/mock.epub"
                }
            });
        });

        await page.route('**/api/analyze/ch_mock', async route => {
            await route.fulfill({
                json: {
                    chapterId: "ch_mock",
                    results: [
                        { text: "Once upon a time...", speaker: "Narrator", emotion: "calm" },
                        { text: "Hello world!", speaker: "Hero", emotion: "happy" }
                    ]
                }
            });
        });

        await page.route('**/api/voices/list', async route => {
            await route.fulfill({ json: { voices: [] } });
        });

        await page.route('**/api/characters', async route => {
            await route.fulfill({ json: { mapping: {} } });
        });

        // 2. Go to Home
        await page.goto('/');

        // 3. Toggle Mock LLM in Settings
        await expect(page.getByText('Settings').first()).toBeVisible();

        const mockCheckbox = page.locator('#mock_llm');
        await mockCheckbox.check();
        await expect(mockCheckbox).toBeChecked();

        // Save
        await page.getByRole('button', { name: 'Save Settings' }).click();
        await expect(page.getByText('Saved successfully!')).toBeVisible();

        // 4. Upload Dummy EPUB
        const buffer = Buffer.from('dummy epub content');
        const fileInput = page.locator('input[type="file"][accept=".epub"]');
        await fileInput.setInputFiles({
            name: 'test.epub',
            mimeType: 'application/epub+zip',
            buffer: buffer
        });

        // 5. Select Chapter
        await expect(page.getByText('Mock Chapter 1')).toBeVisible();
        await page.getByText('Mock Chapter 1').click();

        // 6. Trigger Analysis
        await page.waitForTimeout(1000);
        await expect(page.getByRole('heading', { name: 'Mock Chapter 1' })).toBeVisible();
        await expect(page.getByRole('button', { name: 'Start Analysis' })).toBeVisible();

        await page.getByRole('button', { name: 'Start Analysis' }).click();

        // 7. Verify Results
        // Wait for the success indicator first
        await expect(page.getByText('Analysis Complete')).toBeVisible();
        await expect(page.getByText('Once upon a time...')).toBeVisible();
        await expect(page.getByText('Hero')).toBeVisible();
    });
});
