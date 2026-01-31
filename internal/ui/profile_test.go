package ui

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"testing"
	"time"
)

func BenchmarkNewHome(b *testing.B) {
	os.Setenv("AGENTDECK_PROFILE", "_test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewHome()
	}
}

func TestNewHomeTrace(t *testing.T) {
	os.Setenv("AGENTDECK_PROFILE", "_test")

	f, _ := os.Create("trace.out")
	defer f.Close()
	trace.Start(f)
	defer trace.Stop()

	start := time.Now()
	h := NewHome()
	elapsed := time.Since(start)

	fmt.Printf("NewHome took: %v\n", elapsed)
	if h != nil {
		fmt.Println("NewHome completed successfully")
	}
}

func TestNewHomeCPU(t *testing.T) {
	os.Setenv("AGENTDECK_PROFILE", "_test")

	f, _ := os.Create("cpu.prof")
	defer f.Close()
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	for i := 0; i < 100; i++ {
		h := NewHome()
		if h == nil {
			t.Fatal("NewHome returned nil")
		}
	}

	runtime.GC()
	mf, _ := os.Create("mem.prof")
	defer mf.Close()
	pprof.WriteHeapProfile(mf)
}
