package webspider

import (
	//"github.com/docker/docker/daemon/logger"
	"runtime"
	"testing"
)

//// 测试Docker镜像仓库列表爬取
//func TestFetchDockerMirrorList(t *testing.T) {
//	handler := NewDockerMirrorListHandler("")
//
//	// 调用FetchDockerMirrorList方法
//	ok := handler.FetchList()
//	if !ok {
//		t.Error("Failed to fetch Docker mirror list")
//		return
//	}
//	// 检查是否有数据
//	if len(handler.List) == 0 {
//		t.Error("No Docker mirrors found")
//		return
//	}
//	// 打印结果
//	for _, mirror := range handler.list {
//		t.Logf("Docker Mirror: %s", mirror)
//	}
//}

func TestFetchOs(t *testing.T) {
	t.Logf("Current OS: %s", runtime.GOOS) // darwin是macOS的核心
}

//// 基准测试 - 测量速度和内存使用
//func BenchmarkFetchDockerMirrorList(b *testing.B) {
//	handler := NewDockerMirrorListHandler("")
//
//	// 内存分析准备
//	var memStatsBefore, memStatsAfter runtime.MemStats
//	runtime.ReadMemStats(&memStatsBefore)
//
//	b.ResetTimer()   // 重置计时器，排除初始化时间
//	b.ReportAllocs() // 报告内存分配情况
//
//	for i := 0; i < b.N; i++ {
//		ok := handler.FetchList()
//		if !ok {
//			b.Fatal("Failed to fetch Docker mirror list")
//		}
//
//		if len(handler.list) == 0 {
//			b.Fatal("No Docker mirrors found")
//		}
//	}
//
//	// 内存使用分析
//	runtime.ReadMemStats(&memStatsAfter)
//	b.ReportMetric(float64(memStatsAfter.TotalAlloc-memStatsBefore.TotalAlloc)/1024, "KB/op")
//	b.ReportMetric(float64(memStatsAfter.Mallocs-memStatsBefore.Mallocs), "allocs/op")
//}
//
//// 并发压力测试
//func TestFetchDockerMirrorList_Concurrent(t *testing.T) {
//	const (
//		concurrency = 100              // 并发数
//		iterations  = 100              // 每个goroutine执行次数
//		timeout     = 30 * time.Second // 超时时间
//	)
//
//	var wg sync.WaitGroup
//	wg.Add(concurrency)
//
//	// 监控goroutine
//	done := make(chan bool)
//	go func() {
//		wg.Wait()
//		close(done)
//	}()
//
//	// 记录开始时间
//	start := time.Now()
//
//	// 启动并发测试
//	for i := 0; i < concurrency; i++ {
//		go func(id int) {
//			defer wg.Done()
//
//			for j := 0; j < iterations; j++ {
//				handler := NewDockerMirrorListHandler("")
//				ok := handler.FetchList()
//				if !ok {
//					t.Errorf("Goroutine %d failed on iteration %d", id, j)
//					return
//				}
//
//				if len(handler.list) == 0 {
//					t.Errorf("Goroutine %d got empty list on iteration %d", id, j)
//					return
//				}
//
//				// 模拟一些处理延迟
//				time.Sleep(10 * time.Millisecond)
//			}
//		}(i)
//	}
//
//	// 等待完成或超时
//	select {
//	case <-done:
//		t.Logf("Concurrent test completed in %v", time.Since(start))
//	case <-time.After(timeout):
//		t.Fatalf("Test timed out after %v", timeout)
//	}
//
//	// 打印内存统计
//	var m runtime.MemStats
//	runtime.ReadMemStats(&m)
//	t.Logf("Memory stats: Alloc=%.2fMB, TotalAlloc=%.2fMB, Sys=%.2fMB, NumGC=%d",
//		float64(m.Alloc)/1024/1024,
//		float64(m.TotalAlloc)/1024/1024,
//		float64(m.Sys)/1024/1024,
//		m.NumGC)
//}
//
//// 综合性能测试 - 包含资源监控
//func TestFetchDockerMirrorList_Performance(t *testing.T) {
//	// 1. 预热测试
//	t.Run("Warmup", func(t *testing.T) {
//		handler := NewDockerMirrorListHandler("")
//		ok := handler.FetchList()
//		if !ok {
//			t.Fatal("Warmup failed")
//		}
//	})
//
//	// 2. 单次执行耗时测试
//	t.Run("SingleExecution", func(t *testing.T) {
//		start := time.Now()
//		handler := NewDockerMirrorListHandler("")
//		ok := handler.FetchList()
//		elapsed := time.Since(start)
//
//		if !ok {
//			t.Fatal("Single execution failed")
//		}
//
//		t.Logf("Single execution took %v", elapsed)
//		t.Logf("Fetched %d mirrors", len(handler.list))
//	})
//
//	// 3. 连续执行稳定性测试
//	t.Run("ContinuousExecution", func(t *testing.T) {
//		const iterations = 100
//		var totalTime time.Duration
//		var successCount int
//
//		for i := 0; i < iterations; i++ {
//			start := time.Now()
//			handler := NewDockerMirrorListHandler("")
//			ok := handler.FetchList()
//			elapsed := time.Since(start)
//
//			if ok && len(handler.list) > 0 {
//				successCount++
//				totalTime += elapsed
//			}
//
//			// 每隔10次打印进度
//			if i%10 == 0 {
//				t.Logf("Completed %d/%d iterations", i+1, iterations)
//			}
//		}
//
//		avgTime := totalTime / time.Duration(successCount)
//		t.Logf("Success rate: %d/%d (%.1f%%)", successCount, iterations,
//			float64(successCount)/float64(iterations)*100)
//		t.Logf("Average execution time: %v", avgTime)
//	})
//
//	// 4. 内存泄漏检测
//	t.Run("MemoryLeakCheck", func(t *testing.T) {
//		const iterations = 1000
//		var baseAlloc uint64
//
//		// 获取基础内存使用
//		runtime.GC()
//		var m1 runtime.MemStats
//		runtime.ReadMemStats(&m1)
//		baseAlloc = m1.HeapAlloc
//
//		for i := 0; i < iterations; i++ {
//			handler := NewDockerMirrorListHandler("")
//			_ = handler.FetchList()
//
//			// 每隔100次强制GC
//			if i%100 == 0 {
//				runtime.GC()
//			}
//		}
//
//		// 获取最终内存使用
//		runtime.GC()
//		var m2 runtime.MemStats
//		runtime.ReadMemStats(&m2)
//
//		// 计算内存增长
//		memoryGrowth := int64(m2.HeapAlloc - baseAlloc)
//		t.Logf("Memory growth after %d iterations: %.2fKB",
//			iterations, float64(memoryGrowth)/1024)
//
//		// 设置内存增长阈值（根据实际情况调整）
//		if memoryGrowth > 2*1024*1024 { // 2MB
//			t.Errorf("Possible memory leak: growth > 2MB")
//		}
//	})
//}

//// 并行基准测试
//func BenchmarkFetchDockerMirrorList_Parallel(b *testing.B) {
//	handler := NewDockerMirrorListHandler("")
//
//	b.RunParallel(func(pb *testing.PB) {
//		for pb.next() {
//			ok := handler.FetchList()
//			if !ok {
//				b.Fatal("Failed to fetch Docker mirror list")
//			}
//
//			if len(handler.list) == 0 {
//				b.Fatal("No Docker mirrors found")
//			}
//		}
//	})
//}
