package epub

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
)

type Chapter struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type Reader struct {
	path string
}

// XML Structs for Parsing
type Container struct {
	Rootfiles struct {
		Rootfile []struct {
			FullPath  string `xml:"full-path,attr"`
			MediaType string `xml:"media-type,attr"`
		} `xml:"rootfile"`
	} `xml:"rootfiles"`
}

type Package struct {
	Metadata struct {
		Title string `xml:"title"`
	} `xml:"metadata"`
	Manifest struct {
		Item []struct {
			ID        string `xml:"id,attr"`
			Href      string `xml:"href,attr"`
			MediaType string `xml:"media-type,attr"`
		} `xml:"item"`
	} `xml:"manifest"`
	Spine struct {
		ItemRef []struct {
			IDRef string `xml:"idref,attr"`
		} `xml:"itemref"`
	} `xml:"spine"`
}

func NewReader(path string) (*Reader, error) {
	return &Reader{path: path}, nil
}

func (r *Reader) GetChapters() ([]Chapter, error) {
	z, err := zip.OpenReader(r.path)
	if err != nil {
		return nil, err
	}
	defer z.Close()

	// 1. Find the OPF file via META-INF/container.xml
	containerFile, err := findFileInZip(z, "META-INF/container.xml")
	if err != nil {
		return nil, fmt.Errorf("invalid epub: no container.xml")
	}

	var container Container
	if err := decodeXML(containerFile, &container); err != nil {
		return nil, fmt.Errorf("failed to parse container.xml: %v", err)
	}

	if len(container.Rootfiles.Rootfile) == 0 {
		return nil, fmt.Errorf("invalid epub: no rootfile found")
	}

	opfPath := container.Rootfiles.Rootfile[0].FullPath

	// 2. Parse the OPF file
	opfFile, err := findFileInZip(z, opfPath)
	if err != nil {
		return nil, fmt.Errorf("opf file not found: %s", opfPath)
	}

	var pkg Package
	if err := decodeXML(opfFile, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse opf: %v", err)
	}

	// 3. Build lookup map for manifest items (ID -> Href)
	manifestMap := make(map[string]string)
	for _, item := range pkg.Manifest.Item {
		manifestMap[item.ID] = item.Href
	}

	// 4. Iterate Spine to get chapters in order
	var chapters []Chapter
	opfDir := filepath.Dir(opfPath) // Paths in OPF are relative to OPF location

	count := 1
	for _, itemRef := range pkg.Spine.ItemRef {
		href, ok := manifestMap[itemRef.IDRef]
		if !ok {
			continue
		}

		// Resolve path relative to OPF
		// "OEBPS/content.opf" dir is "OEBPS". Href "Text/chap1.xhtml" -> "OEBPS/Text/chap1.xhtml"
		fullPath := href
		if opfDir != "." {
			// Simple path join, but ensure forward slashes for zip
			fullPath = filepath.ToSlash(filepath.Join(opfDir, href))
		}

		// Find file in zip
		f, err := findFileInZip(z, fullPath)
		if err != nil {
			fmt.Printf("Warning: missing file %s\n", fullPath)
			continue
		}

		// Extract content
		content, err := extractText(f)
		if err != nil {
			continue
		}

		// Use count as ID to ensure order is preserved in frontend
		if len(strings.TrimSpace(content)) > 10 {
			chapters = append(chapters, Chapter{
				ID:      fmt.Sprintf("ch_%03d", count),
				Title:   fmt.Sprintf("Chapter %d", count),
				Content: content,
			})
			count++
		}
	}

	return chapters, nil
}

func findFileInZip(z *zip.ReadCloser, name string) (*zip.File, error) {
	for _, f := range z.File {
		// Zip headers usually use forward slash. Windows paths might be mixed if created poorly.
		// We normalize name to match zip standard.
		if f.Name == name || f.Name == strings.ReplaceAll(name, "\\", "/") {
			return f, nil
		}
	}
	return nil, fmt.Errorf("file not found")
}

func decodeXML(f *zip.File, target interface{}) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	return xml.NewDecoder(rc).Decode(target)
}

func extractText(f *zip.File) (string, error) {
	rc, err := f.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	doc, err := html.Parse(rc)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	var walker func(*html.Node)
	walker = func(n *html.Node) {
		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				sb.WriteString(text + "\n")
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walker(c)
		}
	}
	walker(doc)
	return sb.String(), nil
}
