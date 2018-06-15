package amdconfig

/*
#cgo CFLAGS: -Icl -I.
#cgo !darwin LDFLAGS: -lOpenCL
#cgo darwin LDFLAGS: -framework OpenCL

#include "topology.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/gurupras/minerconfig/pcie"
	cl "github.com/rainliu/gocl/cl"
)

func GetDeviceTopology(deviceId cl.CL_device_id) (*pcie.Topology, error) {
	var ret C.struct_topology
	dptr := unsafe.Pointer(&deviceId)
	rptr := unsafe.Pointer(&ret)
	C.GetTopology(dptr, rptr)
	topology := &pcie.Topology{
		int(ret.bus),
		int(ret.device),
		int(ret.function),
	}
	return topology, nil
}

func FindIndexMatchingTopology(topology *pcie.Topology) (int, error) {
	numPlatforms := getNumPlatforms()
	for platformIdx := 0; platformIdx < int(numPlatforms); platformIdx++ {
		amdDevices := getAMDDevices(platformIdx)
		for _, ctx := range amdDevices {
			devTopology, _ := GetDeviceTopology(ctx.DeviceID)
			if *devTopology == *topology {
				return ctx.DeviceIndex, nil
			}
		}
	}
	return -1, fmt.Errorf("Failed to find an OpenCL device matching topology: %v\n", *topology)
}
