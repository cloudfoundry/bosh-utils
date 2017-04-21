package httpclient_test

import "syscall"

const keepaliveIntervalPlatformIndependent = syscall.TCP_KEEPINTVL
const keepaliveGetsockoptPlatformIndependent = 0x1