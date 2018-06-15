#include "topology.h"
#include <stdio.h>
#include <assert.h>

inline void GetTopology(void *dptr, void *rptr)
{
  cl_device_id *deviceId = (cl_device_id *) dptr;
  struct topology *ret = (struct topology *) rptr;

  cl_device_topology_amd topology;
  int status = clGetDeviceInfo (*deviceId, CL_DEVICE_TOPOLOGY_AMD,
      sizeof(cl_device_topology_amd), &topology, NULL);

  if(status != CL_SUCCESS) {
      // TODO: Handle error
  }

  if (topology.raw.type == CL_DEVICE_TOPOLOGY_TYPE_PCIE_AMD) {
    ret->bus = (int) topology.pcie.bus;
    ret->device = (int) topology.pcie.device;
    ret->function = (int) topology.pcie.function;
  }
}
