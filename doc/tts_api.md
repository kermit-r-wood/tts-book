# IndexTTS API Documentation

1. Confirm that you have cURL installed on your system.

```bash
$ curl --version
```

2. Find the API endpoint below corresponding to your desired function in the app. Copy the code snippet, replacing the placeholder values with your own input data.

## API name: `/on_example_click`

```bash
curl -X POST http://localhost:7860/gradio_api/call/on_example_click -s -H "Content-Type: application/json" -d '{
	"data": [
							0
	]}' \
	| awk -F'"' '{ print $4}'  \
	| read EVENT_ID; curl -N http://localhost:7860/gradio_api/call/on_example_click/$EVENT_ID
```

Accepts 1 parameter:
- `[0]` any Required: The input value that is provided in the "Examples" Dataset component.

Returns list of 14 elements:
- `[0]`: The output value that appears in the "Voice Reference" Audio component.
- `[1]` string: The output value that appears in the "Emotion control method" Radio component.
- `[2]` string: The output value that appears in the "Text" Textbox component.
- `[3]`: The output value that appears in the "Upload emotion reference audio" Audio component.
- `[4]` number: The output value that appears in the "Emotion control weight" Slider component.
- `[5]` string: The output value that appears in the "Emotion description" Textbox component.
- `[6]` number: The output value that appears in the "Happy" Slider component.
- `[7]` number: The output value that appears in the "Angry" Slider component.
- `[8]` number: The output value that appears in the "Sad" Slider component.
- `[9]` number: The output value that appears in the "Afraid" Slider component.
- `[10]` number: The output value that appears in the "Disgusted" Slider component.
- `[11]` number: The output value that appears in the "Melancholic" Slider component.
- `[12]` number: The output value that appears in the "Surprised" Slider component.
- `[13]` number: The output value that appears in the "Calm" Slider component.

## API name: `/on_method_change`

```bash
curl -X POST http://localhost:7860/gradio_api/call/on_method_change -s -H "Content-Type: application/json" -d '{
	"data": [
							"Same as the voice reference"
	]}' \
	| awk -F'"' '{ print $4}'  \
	| read EVENT_ID; curl -N http://localhost:7860/gradio_api/call/on_method_change/$EVENT_ID
```

Accepts 1 parameter:
- `[0]` string Required: The input value that is provided in the "Emotion control method" Radio component.

Returns 1 element.

## API name: `/on_experimental_change`

```bash
curl -X POST http://localhost:7860/gradio_api/call/on_experimental_change -s -H "Content-Type: application/json" -d '{
	"data": [
							true,
							"Same as the voice reference"
	]}' \
	| awk -F'"' '{ print $4}'  \
	| read EVENT_ID; curl -N http://localhost:7860/gradio_api/call/on_experimental_change/$EVENT_ID
```

Accepts 2 parameters:
- `[0]` boolean Required: The input value that is provided in the "Show experimental features" Checkbox component.
- `[1]` string Required: The input value that is provided in the "Emotion control method" Radio component.

Returns list of 2 elements:
- `[0]` string: The output value that appears in the "Emotion control method" Radio component.
- `[1]`: The output value that appears in the "Examples" Dataset component.

## API name: `/on_glossary_checkbox_change` 
Controls the visibility of the glossary.

```bash
curl -X POST http://localhost:7860/gradio_api/call/on_glossary_checkbox_change -s -H "Content-Type: application/json" -d '{
	"data": [
							true
	]}' \
	| awk -F'"' '{ print $4}'  \
	| read EVENT_ID; curl -N http://localhost:7860/gradio_api/call/on_glossary_checkbox_change/$EVENT_ID
```

Accepts 1 parameter:
- `[0]` boolean Required: The input value that is provided in the "Enable custom term pronunciations" Checkbox component.

Returns 1 element.

## API name: `/on_input_text_change`

```bash
curl -X POST http://localhost:7860/gradio_api/call/on_input_text_change -s -H "Content-Type: application/json" -d '{
	"data": [
							"Hello!!",
							20
	]}' \
	| awk -F'"' '{ print $4}'  \
	| read EVENT_ID; curl -N http://localhost:7860/gradio_api/call/on_input_text_change/$EVENT_ID
```

Accepts 2 parameters:
- `[0]` string Required: The input value that is provided in the "Text" Textbox component.
- `[1]` number Required: The input value that is provided in the "Max tokens per generation segment" Slider component.

Returns 1 element: The output value that appears in the "value_83" Dataframe component.

## API name: `/on_input_text_change_1`

```bash
curl -X POST http://localhost:7860/gradio_api/call/on_input_text_change_1 -s -H "Content-Type: application/json" -d '{
	"data": [
							"Hello!!",
							20
	]}' \
	| awk -F'"' '{ print $4}'  \
	| read EVENT_ID; curl -N http://localhost:7860/gradio_api/call/on_input_text_change_1/$EVENT_ID
```

Accepts 2 parameters:
- `[0]` string Required: The input value that is provided in the "Text" Textbox component.
- `[1]` number Required: The input value that is provided in the "Max tokens per generation segment" Slider component.

Returns 1 element: The output value that appears in the "value_83" Dataframe component.

## API name: `/update_prompt_audio`

```bash
curl -X POST http://localhost:7860/gradio_api/call/update_prompt_audio -s -H "Content-Type: application/json" -d '{
	"data": [
	]}' \
	| awk -F'"' '{ print $4}'  \
	| read EVENT_ID; curl -N http://localhost:7860/gradio_api/call/update_prompt_audio/$EVENT_ID
```

Accepts 0 parameters.
Returns 1 element.

## API name: `/on_add_glossary_term`
Add term to vocabulary and auto-save.

```bash
curl -X POST http://localhost:7860/gradio_api/call/on_add_glossary_term -s -H "Content-Type: application/json" -d '{
	"data": [
							"Hello!!",
							"Hello!!",
							"Hello!!"
	]}' \
	| awk -F'"' '{ print $4}'  \
	| read EVENT_ID; curl -N http://localhost:7860/gradio_api/call/on_add_glossary_term/$EVENT_ID
```

Accepts 3 parameters:
- `[0]` string Required: The input value that is provided in the "Term" Textbox component.
- `[1]` string Required: The input value that is provided in the "Chinese pronunciation" Textbox component.
- `[2]` string Required: The input value that is provided in the "English pronunciation" Textbox component.

Returns 1 element: string (The output value that appears in the "value_57" Markdown component).

## API name: `/on_demo_load`
Reload glossary data on page load.

```bash
curl -X POST http://localhost:7860/gradio_api/call/on_demo_load -s -H "Content-Type: application/json" -d '{
	"data": [
	]}' \
	| awk -F'"' '{ print $4}'  \
	| read EVENT_ID; curl -N http://localhost:7860/gradio_api/call/on_demo_load/$EVENT_ID
```

Accepts 0 parameters.
Returns 1 element: string (The output value that appears in the "value_57" Markdown component).

## API name: `/gen_single`

```bash
curl -X POST http://localhost:7860/gradio_api/call/gen_single -s -H "Content-Type: application/json" -d '{
	"data": [
							"Same as the voice reference",
							{"path":"https://github.com/gradio-app/gradio/raw/main/test/test_files/audio_sample.wav","meta":{"_type":"gradio.FileData"}},
							"Hello!!",
							{"path":"https://github.com/gradio-app/gradio/raw/main/test/test_files/audio_sample.wav","meta":{"_type":"gradio.FileData"}},
							0,
							0,
							0,
							0,
							0,
							0,
							0,
							0,
							0,
							"Hello!!",
							true,
							20,
							true,
							0,
							0,
							0.1,
							-2,
							1,
							0.1,
							50
	]}' \
	| awk -F'"' '{ print $4}'  \
	| read EVENT_ID; curl -N http://localhost:7860/gradio_api/call/gen_single/$EVENT_ID
```

Accepts 24 parameters:
- `[0]` string Required: "Emotion control method" (e.g., "Same as the voice reference", "Use emotion vectors").
- `[1]` any Required: "Voice Reference" Audio component (FileData object).
- `[2]` string Required: "Text" Textbox component.
- `[3]` any Required: "Upload emotion reference audio" Audio component (FileData object).
- `[4]` number Required: "Emotion control weight" Slider component.
- `[5]` number Required: "Happy" Slider component.
- `[6]` number Required: "Angry" Slider component.
- `[7]` number Required: "Sad" Slider component.
- `[8]` number Required: "Afraid" Slider component.
- `[9]` number Required: "Disgusted" Slider component.
- `[10]` number Required: "Melancholic" Slider component.
- `[11]` number Required: "Surprised" Slider component.
- `[12]` number Required: "Calm" Slider component.
- `[13]` string Required: "Emotion description" Textbox component.
- `[14]` boolean Required: "Randomize emotion sampling" Checkbox component.
- `[15]` number Required: "Max tokens per generation segment" Slider component.
- `[16]` boolean Required: "do_sample" Checkbox component.
- `[17]` number Required: "top_p" Slider component.
- `[18]` number Required: "top_k" Slider component.
- `[19]` number Required: "temperature" Slider component.
- `[20]` number Required: "length_penalty" Number component.
- `[21]` number Required: "num_beams" Slider component.
- `[22]` number Required: "repetition_penalty" Number component.
- `[23]` number Required: "max_mel_tokens" Slider component.

Returns 1 element: The output value that appears in the "Synthesis Result" Audio component.