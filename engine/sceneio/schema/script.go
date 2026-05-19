package schema

type CameraScript struct {
	ID           string      `json:"id"`
	Type         string      `json:"type"`
	Position     []float64   `json:"position"`
	LookAt       []float64   `json:"look_at"`
	Direction    []float64   `json:"direction"`
	Up           []float64   `json:"up"`
	Width        int         `json:"width"`
	Height       int         `json:"height"`
	Widths       []int       `json:"widths"`
	FieldOfView  float64     `json:"field_of_view"`
	FieldOfViews []float64   `json:"field_of_views"`
	Coordinates  [][]float64 `json:"coordinates"`
	AspectRatio  float64     `json:"aspect_ratio"`
	Ortho        bool        `json:"ortho"`
}

type RenderScript struct {
	Dimension         int     `json:"dimension"`
	Samples           int64   `json:"samples"`
	ThreadNum         int     `json:"thread_num"`
	CameraIndex       int     `json:"camera_index"`
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
	WorkingSpace      string  `json:"working_space"`
}

type Script struct {
	Materials []map[string]interface{}          `json:"materials"`
	Media     map[string]map[string]interface{} `json:"media"`
	Objects   []map[string]interface{}          `json:"objects"`
	Cameras   []CameraScript                    `json:"cameras"`
	Render    RenderScript                      `json:"render"`
}
