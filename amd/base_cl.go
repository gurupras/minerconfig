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
	"math"
	"strings"
	"unsafe"

	"github.com/gurupras/go-cryptonight-miner/gpu-miner/gpucontext"
	cl "github.com/rainliu/gocl/cl"
	log "github.com/sirupsen/logrus"
)

func err_to_str(ret cl.CL_int) string {
	result := C.err_to_str(C.int(ret))
	return C.GoString(result)
}

func getNumPlatforms() cl.CL_uint {
	var (
		count cl.CL_uint = 0
		ret   cl.CL_int
	)

	if ret = cl.CLGetPlatformIDs(0, nil, &count); ret != cl.CL_SUCCESS {
		log.Errorf("Failed to call clGetPlatformIDs: %v", err_to_str(ret))
	}
	return count
}

func getDeviceMaxComputeUnits(id cl.CL_device_id) cl.CL_uint {
	count := 0
	var info interface{}
	cl.CLGetDeviceInfo(id, cl.CL_DEVICE_MAX_COMPUTE_UNITS, cl.CL_size_t(unsafe.Sizeof(count)), &info, nil)
	return info.(cl.CL_uint)
}

func getDeviceInfoBytes(deviceId cl.CL_device_id, info cl.CL_device_info, size cl.CL_size_t) ([]byte, error) {
	var ret interface{}
	if err := cl.CLGetDeviceInfo(deviceId, info, size, &ret, nil); err != cl.CL_SUCCESS {
		return nil, fmt.Errorf("Failed to get device info: %v", err_to_str(err))
	}
	return []byte(ret.(string)), nil
}

func getAMDDevices(index int) (contexts []*gpucontext.GPUContext) {
	numPlatforms := getNumPlatforms()

	platforms := make([]cl.CL_platform_id, numPlatforms)
	cl.CLGetPlatformIDs(numPlatforms, platforms, nil)

	var numDevices cl.CL_uint
	cl.CLGetDeviceIDs(platforms[index], cl.CL_DEVICE_TYPE_GPU, 0, nil, &numDevices)

	deviceList := make([]cl.CL_device_id, numDevices)
	cl.CLGetDeviceIDs(platforms[index], cl.CL_DEVICE_TYPE_GPU, numDevices, deviceList, nil)

	contexts = make([]*gpucontext.GPUContext, 0)
	for i := cl.CL_uint(0); i < numDevices; i++ {
		data, err := getDeviceInfoBytes(deviceList[i], cl.CL_DEVICE_VENDOR, 256)
		if err != nil {
			log.Errorf("Failed to get CL_DEVICE_VENDOR: %v", err)
			continue
		}
		str := string(data)
		if !strings.Contains(str, "Advanced Micro Devices") {
			continue
		}

		ctx := gpucontext.New(int(i), 0, 0)
		ctx.DeviceID = deviceList[i]
		ctx.ComputeUnits = getDeviceMaxComputeUnits(ctx.DeviceID)

		var (
			maxMem  cl.CL_ulong
			freeMem cl.CL_ulong
			mmIface interface{}
			fmIface interface{}
		)
		cl.CLGetDeviceInfo(ctx.DeviceID, cl.CL_DEVICE_MAX_MEM_ALLOC_SIZE, cl.CL_size_t(4), &mmIface, nil)
		cl.CLGetDeviceInfo(ctx.DeviceID, cl.CL_DEVICE_GLOBAL_MEM_SIZE, cl.CL_size_t(4), &fmIface, nil)
		// log.Infof("Types: maxMem: %t  freeMem: %t", maxMem, freeMem)
		maxMem = mmIface.(cl.CL_ulong)
		freeMem = fmIface.(cl.CL_ulong)
		ctx.FreeMemory = cl.CL_ulong(math.Min(float64(maxMem), float64(freeMem)))

		friendlyNameBytes, err := getDeviceInfoBytes(deviceList[i], cl.CL_DEVICE_NAME, 256)
		if err != nil {
			log.Errorf("Failed to get device name: %v", err)
			continue
		}
		ctx.Name = string(friendlyNameBytes)
		log.Debugf("OpenCL GPU: %v, cpu: %d", ctx.Name, ctx.ComputeUnits)
		contexts = append(contexts, ctx)
	}
	return
}
