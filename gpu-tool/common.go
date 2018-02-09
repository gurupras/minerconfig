package gputool

import "fmt"

type GPUToolType string

const (
	GPU_TOOL_MSI_AB          GPUToolType = "MSI-AB"
	GPU_TOOL_OVERDRIVE_NTOOL GPUToolType = "ODNT"
)

type GPUToolInterface interface {
	Run(path string, args map[string]interface{}) error
}

func ParseGPUTool(toolType GPUToolType) (GPUToolInterface, error) {
	switch toolType {
	case GPU_TOOL_MSI_AB:
		return &MSIAfterBurner{}, nil
	case GPU_TOOL_OVERDRIVE_NTOOL:
		return &OverdriveNTool{}, nil
	default:
		return nil, fmt.Errorf("Unimplemented GPUTool: %v", toolType)
	}
}
