#ifndef __BASE_H_
#define __BASE_H_

#ifdef __APPLE__
#include "OpenCL/opencl.h"
#else
#include "CL/opencl.h"
#include "CL/cl_ext.h"
#endif

char* err_to_str(int ret);
#endif
