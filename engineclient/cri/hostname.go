// Copyright 2023 Harald Albrecht.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cri

import (
	"os"
	"regexp"
	"runtime"
	"strconv"

	"golang.org/x/sys/unix"
)

var onlyHostnameRegexp = regexp.MustCompile(`[^a-zA-Z0-9_\-.]+`)

// hostname returns the UTS hostname for the process with the specified PID,
// otherwise falling back to /etc/hostname. If PID is zero, then the current
// process' PID is assumed.
func hostname(pid int) string {
	// our first attempt is to fetch the UTS host name...
	var hostname string
	if pid == 0 {
		hostname, _ = os.Hostname()
	} else {
		visitUTS(pid, func() { hostname, _ = os.Hostname() })
	}
	if hostname != "" {
		return hostname
	}
	// ...and that didn't went well, so now we try to read /etc/hosts.
	if pid == 0 {
		pid = os.Getpid()
	}
	octets, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/root/etc/hostname")
	if err != nil {
		return ""
	}
	return onlyHostnameRegexp.ReplaceAllString(string(octets), "")
}

// visitUTS calls the specified function fn in the UTS context of the process
// identified by its PID. In case of errors, it may not have called fn.
//
// Doing the UTS namespace switching "by hand" avoids a dependency on the lxkns
// module.
func visitUTS(pid int, fn func()) {
	origUTSfd, err := unix.Open("/proc/self/ns/uts", unix.O_RDONLY, 0)
	if err != nil {
		return
	}
	defer unix.Close(origUTSfd)
	newUTSfd, err := unix.Open("/proc/"+strconv.Itoa(pid)+"/ns/uts", unix.O_RDONLY, 0)
	if err != nil {
		return
	}
	defer unix.Close(newUTSfd)

	done := make(chan struct{})
	go func() {
		defer close(done)
		runtime.LockOSThread()
		if err := unix.Setns(newUTSfd, 0); err != nil {
			runtime.UnlockOSThread()
			return
		}
		fn()
		if err := unix.Setns(origUTSfd, 0); err != nil {
			// In case we cannot switch back for whatever reason, we do not
			// unlock the OS-level thread locked to this Go routine so that the
			// Go runtime then needs to throw away the thread as soon as our Go
			// routine finishes. Or, in the case of M0, the thread gets "wedged"
			// ... but callers should avoid this by locking M0 to the process'
			// initial go routine, so this go routine here can never get its
			// dirty paws on M0.
			return
		}
		runtime.UnlockOSThread()
	}()
	<-done // wait for fn to be called or something gone terribly wrong.
}
