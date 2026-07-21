package schema

import "encoding/json"

type StudioScript struct {
	Includes  []string                          `json:"includes"`
	Materials []map[string]interface{}          `json:"materials"`
	Media     map[string]map[string]interface{} `json:"media"`
	Objects   []map[string]interface{}          `json:"objects"`
	Cameras   []StudioCameraScript              `json:"cameras"`
	Render    StudioRenderScript                `json:"render"`
	Geometry  map[string]interface{}            `json:"geometry"`
	Renders   []StudioRenderScript              `json:"renders"`
}

type StudioRenderScript struct {
	Dimension         int     `json:"dimension"`
	Samples           int64   `json:"samples"`
	ThreadNum         int     `json:"thread_num"`
	CameraIndex       int     `json:"camera_index"`
	CameraIndexSet    bool    `json:"-"`
	Width             int     `json:"width"`
	Height            int     `json:"height"`
	OutputImage       string  `json:"output_image"`
	OutputFilm        string  `json:"output_film"`
	ResumeFilm        string  `json:"resume_film"`
	Exposure          float64 `json:"exposure"`
	ToneMapping       string  `json:"tone_mapping"`
	Gamma             float64 `json:"gamma"`
	SpectrumMode      string  `json:"spectrum_mode"`
	WavelengthSamples int     `json:"wavelength_samples"`
	ColorSpace        string  `json:"color_space"`
	FilmColorSpace    string  `json:"working_space"`
}

func (r *StudioRenderScript) UnmarshalJSON(data []byte) error {
	type renderScript StudioRenderScript
	var decoded renderScript
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	decoded.CameraIndexSet = raw["camera_index"] != nil
	*r = StudioRenderScript(decoded)
	return nil
}

type StudioCameraScript struct {
	ID           string      `json:"id"`
	Type         string      `json:"type"`
	Position     []float64   `json:"position"`
	LookAt       []float64   `json:"look_at"`
	Direction    []float64   `json:"direction"`
	Up           []float64   `json:"up"`
	Widths       []int       `json:"widths"`
	FieldOfView  float64     `json:"field_of_view"`
	FieldOfViews []float64   `json:"field_of_views"`
	Coordinates  [][]float64 `json:"coordinates"`
	AspectRatio  float64     `json:"aspect_ratio"`
	Ortho        bool        `json:"ortho"`
}

type IntermediateScript struct {
	Studio    StudioMetadata                    `json:"_studio"`
	Materials []map[string]interface{}          `json:"materials,omitempty"`
	Media     map[string]map[string]interface{} `json:"media,omitempty"`
	Objects   []map[string]interface{}          `json:"objects,omitempty"`
	Cameras   []EngineCameraScript              `json:"cameras,omitempty"`
	Render    map[string]interface{}            `json:"render,omitempty"`
	Geometry  map[string]interface{}            `json:"geometry,omitempty"`
	Renders   []map[string]interface{}          `json:"renders,omitempty"`
}

type StudioMetadata struct {
	Version     string   `json:"version"`
	Source      []string `json:"source"`
	GeneratedAt string   `json:"generated_at"`
	Dimension   int      `json:"dimension"`
}

type EngineCameraScript struct {
	ID           string      `json:"id,omitempty"`
	Type         string      `json:"type,omitempty"`
	Position     []float64   `json:"position,omitempty"`
	Direction    []float64   `json:"direction,omitempty"`
	Up           []float64   `json:"up,omitempty"`
	Widths       []int       `json:"widths,omitempty"`
	FieldOfViews []float64   `json:"field_of_views,omitempty"`
	Coordinates  [][]float64 `json:"coordinates,omitempty"`
	Ortho        bool        `json:"ortho,omitempty"`
}
