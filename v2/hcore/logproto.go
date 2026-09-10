package hcore

import (
	"fmt"
	"os"
	"runtime/pprof"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const asyncLogBufferSize = 1024

type asyncLogEntry struct {
	level LogLevel
	typ   LogType
	time  time.Time
	args  []any
}

var (
	asyncLogOnce    sync.Once
	asyncLogQueue   = make(chan asyncLogEntry, asyncLogBufferSize)
	asyncLogDropped atomic.Uint64
)

func Log(level LogLevel, typ LogType, message ...any) {
	if level < static.logLevel {
		return
	}
	asyncLogOnce.Do(func() {
		go drainAsyncLogs()
	})

	entry := asyncLogEntry{
		level: level,
		typ:   typ,
		time:  time.Now(),
		args:  append([]any(nil), message...),
	}
	select {
	case asyncLogQueue <- entry:
	default:
		asyncLogDropped.Add(1)
	}
}

func drainAsyncLogs() {
	for entry := range asyncLogQueue {
		writeAsyncLog(entry)
	}
}

func writeAsyncLog(entry asyncLogEntry) {
	if dropped := asyncLogDropped.Swap(0); dropped > 0 {
		publishLogMessage(
			LogLevel_WARNING,
			LogType_CORE,
			time.Now(),
			fmt.Sprintf("dropped %d log messages because the async log queue is full", dropped),
		)
	}

	message := fmt.Sprint(entry.args...)
	publishLogMessage(entry.level, entry.typ, entry.time, message)
}

func publishLogMessage(level LogLevel, typ LogType, timestamp time.Time, message string) {
	static.logObserver.Publish(&LogMessage{
		Level:   level,
		Type:    typ,
		Time:    timestamppb.New(timestamp),
		Message: message,
	})
}

func (s *CoreService) LogListener(req *LogRequest, stream grpc.ServerStreamingServer[LogMessage]) error {
	logSub := static.logObserver.Subscribe(1)
	defer static.logObserver.Unsubscribe(logSub)

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case info := <-logSub:
			if info.Level < req.Level {
				continue
			}
			stream.Send(info)
			// case <-time.After(500 * time.Millisecond):
		}
	}
}

func dumpGoroutinesToFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return pprof.Lookup("goroutine").WriteTo(f, 2)
}
