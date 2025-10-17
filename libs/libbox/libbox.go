package main

/*
#include <stdlib.h>
#include <string.h>
#include <stdbool.h>
*/
import "C"
import (
	"context"
	"encoding/json"
	"os"
	"os/user"
	"strconv"
	"unsafe"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/experimental/deprecated"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing/service"
	"github.com/sagernet/sing/service/filemanager"
)

var globalCtx context.Context

type BoxRef struct {
	box    *box.Box
	ctx    context.Context
	cancel context.CancelFunc
}

//export BoxCreate
func BoxCreate(config *C.char) unsafe.Pointer {
	conf := C.GoString(config)

	ctx, cancel := context.WithCancel(globalCtx)

	options := box.Options{
		Context: ctx,
	}
	if err := json.Unmarshal([]byte(conf), &options); err != nil {
		log.ErrorContext(globalCtx, err)
		cancel()
		return nil
	}

	instance, err := box.New(options)
	if err != nil {
		log.ErrorContext(globalCtx, err)
		cancel()
		return nil
	}

	const boxRefSize = C.size_t(unsafe.Sizeof(BoxRef{}))

	boxRefPtr := C.malloc(boxRefSize)
	if boxRefPtr == nil {
		log.ErrorContext(globalCtx, "failed to allocate for box ref")
		cancel()
		return nil
	}
	C.memset(boxRefPtr, C.int(0), boxRefSize)

	boxRef := (*BoxRef)(boxRefPtr)
	boxRef.ctx = ctx
	boxRef.cancel = cancel
	boxRef.box = instance

	return boxRefPtr
}

//export BoxDestroy
func BoxDestroy(boxRefPtr unsafe.Pointer) C.bool {
	if boxRefPtr == nil {
		return C.bool(false)
	}

	boxRef := (*BoxRef)(boxRefPtr)
	if boxRef.cancel != nil {
		boxRef.cancel()
	}

	if boxRef.box == nil {
		return C.bool(false)
	}

	if err := boxRef.box.Close(); err != nil {
		log.ErrorContext(globalCtx, err)
	}

	return C.bool(true)
}

//export BoxStart
func BoxStart(boxRefPtr unsafe.Pointer) C.size_t {
	if boxRefPtr == nil {
		return C.size_t(0)
	}

	boxRef := (*BoxRef)(boxRefPtr)
	if boxRef.box == nil {
		return C.size_t(0)
	}

	// this start function freezes current thread
	// this must get executed inside another goroutine!
	if err := boxRef.box.Start(); err != nil {
		log.ErrorContext(globalCtx, err)
		return C.size_t(0)
	}

	return C.size_t(155)
}

//export PreRun
func PreRun() {
	globalCtx = context.Background()
	sudoUser := os.Getenv("SUDO_USER")
	sudoUID, _ := strconv.Atoi(os.Getenv("SUDO_UID"))
	sudoGID, _ := strconv.Atoi(os.Getenv("SUDO_GID"))
	if sudoUID == 0 && sudoGID == 0 && sudoUser != "" {
		sudoUserObject, _ := user.Lookup(sudoUser)
		if sudoUserObject != nil {
			sudoUID, _ = strconv.Atoi(sudoUserObject.Uid)
			sudoGID, _ = strconv.Atoi(sudoUserObject.Gid)
		}
	}
	if sudoUID > 0 && sudoGID > 0 {
		globalCtx = filemanager.WithDefault(globalCtx, "", "", sudoUID, sudoGID)
	}
	globalCtx = service.ContextWith(globalCtx, deprecated.NewStderrManager(log.StdLogger()))
}

func main() {}
