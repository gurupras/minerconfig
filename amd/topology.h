#ifndef __TOPOLOGY_H_
#define __TOPOLOGY_H_

#include "base.h"

struct topology {
  int bus;
  int device;
  int function;
};

void GetTopology(void *dptr, void *rptr);
#endif
