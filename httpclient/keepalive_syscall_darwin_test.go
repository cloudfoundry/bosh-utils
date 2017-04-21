package httpclient_test

import "syscall"

const keepaliveIntervalPlatformIndependent = syscall.TCP_KEEPALIVE
const keepaliveGetsockoptPlatformIndependent = syscall.SO_KEEPALIVE