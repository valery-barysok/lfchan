package lfchan

import (
	"flag"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/OneOfOne/lfchan/typed/uint64Chan"
)

var timeCPU = flag.Bool("timeCpu", false, "internal")

func TestLFChan(t *testing.T) {
	ch := New()
	var want, got int
	for i := 0; i < 100; i++ {
		go ch.Send(i, true)
		want += i
	}
	for i := 0; i < 100; i++ {
		v, ok := ch.Recv(true)
		if !ok {
			t.Fatal("!ok")
		}
		got += v.(int)
	}
	if want != got {
		t.Fatalf("wanted %v, got %v", want, got)
	}
}

func TestSelect(t *testing.T) {
	var chs [100]Chan
	for i := range chs {
		chs[i] = New()
	}
	for i := range chs {
		var senders [100]Sender
		for i := range chs {
			senders[i] = chs[i]
		}
		if !SelectSend(false, i, senders[:]...) {
			t.Fatalf("couldn't send %d", i)
		}
	}
	for i := range chs {
		var recvs [100]Receiver
		for i := range chs {
			recvs[i] = chs[i]
		}
		if v, ok := SelectRecv(false, recvs[:]...); !ok || v != i {
			t.Fatalf("wanted %v, got %v", i, v)
		}
	}
}

func TestFIFO(t *testing.T) {
	const N = 1e5
	ch := NewSize(100)

	go func() {
		for i := 0; i < N; i++ {
			ch.Send(i, true)
		}
	}()
	for i := 0; i < N; i++ {
		v, ok := ch.Recv(true)
		if !ok {
			t.Fatal("!ok")
		}
		if v.(int) != i {
			t.Fatalf("wanted %d, got %d", i, v)
		}
	}
}

// needs to run with -count 100 to trigger
func TestLen(t *testing.T) {
	t.Skip("broken / buggy for now")
	var N = 1000 * (runtime.GOMAXPROCS(0) << 4) // otherwise it gets really slow on < 8 cores.
	t.Log(N)
	ch := NewSize(100)
	var wg sync.WaitGroup
	wg.Add(N * 2)

	go func() {
		for i := 0; i < N; i++ {
			go func() {
				v, ok := ch.Recv(true)
				if !ok {
					wg.Done()
					t.Fatal("!ok")
				}
				if ln := ch.Len(); ln < 0 {
					t.Fatalf("ch.Len() == %d: %v", ln, v)
				}
				wg.Done()
			}()
		}
	}()

	go func() {
		for i := 0; i < N; i++ {
			go func(i int) {
				ch.Send(i, true)
				wg.Done()
			}(i)
		}
	}()
	wg.Wait()
	ch.Close()
}

func TestLFCPU(t *testing.T) {
	if !*timeCPU {
		t.SkipNow()
	}
	ch := NewSize(1)
	go func() {
		for i := 0; i < 10; i++ {
			ch.Send(i, true)
			time.Sleep(time.Second)
		}
		ch.Close()
	}()
	for v, ok := ch.Recv(true); ok && v != nil; v, ok = ch.Recv(true) {
		t.Log(v)
	}
}

func TestStdCPU(t *testing.T) {
	if !*timeCPU {
		t.SkipNow()
	}
	ch := make(chan interface{}, 1)
	go func() {
		for i := 0; i < 10; i++ {
			ch <- i
			time.Sleep(time.Second)
		}
		close(ch)
	}()
	for v := range ch {
		t.Log(v)
	}
}

func BenchmarkLFChan(b *testing.B) {
	ch := uint64Chan.NewSize(100)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var cnt uint64
		var total uint64
		for pb.Next() {
			ch.Send(cnt, true)
			v, _ := ch.Recv(true)
			total += v
			cnt++
		}
	})
}

func BenchmarkChan(b *testing.B) {
	ch := make(chan uint64, 100)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var cnt uint64
			var total uint64
			for pb.Next() {
				ch <- cnt
				v := <-ch
				total += v
				cnt++
			}
		}
	})
}
