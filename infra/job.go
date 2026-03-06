package infra

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

// JobStatus 表示基础 Job 的当前状态。
type JobStatus string

const (
	JobPending   JobStatus = "pending"
	JobRunning   JobStatus = "running"
	JobSucceeded JobStatus = "succeeded"
	JobFailed    JobStatus = "failed"
	JobCanceled  JobStatus = "canceled"
)

// Job 表示一个通用的基础任务，带 topic 以便多模块复用。
type Job struct {
	ID         string            `json:"id"`
	Topic      string            `json:"topic"`                // 例如 "cicd", "pipeline", "spider"
	Status     JobStatus         `json:"status"`               // pending / running / succeeded / failed / canceled
	Error      string            `json:"error,omitempty"`      // 失败或取消原因
	Meta       map[string]string `json:"meta,omitempty"`       // 轻量级元数据（repo、branch 等）
	EnqueuedAt time.Time         `json:"enqueued_at"`          // 入队时间
	StartedAt  time.Time         `json:"started_at,omitempty"` // 开始执行时间
	FinishedAt time.Time         `json:"finished_at,omitempty"`
}

// JobFunc 是基础 Job 的执行函数签名。
// ctx 用于超时和取消；当队列关闭或调用方的 ctx 取消时，ctx 会被取消。
// fn 可以根据需要更新 job.Meta 的内容（需自己加锁的话可以在业务层完成）。
type JobFunc func(ctx context.Context, job *Job) error

type jobEntry struct {
	job *Job
	ctx context.Context
	fn  JobFunc
}

// JobQueue 是一个进程内 Job 执行器，带有限制的队列和固定数量的 worker。
// 任务会在后台 goroutine 中执行，可通过 Jobs / Job 方法观测状态。
type JobQueue struct {
	mu   sync.RWMutex
	jobs map[string]*Job

	queue   chan *jobEntry
	workers int

	onError func(job *Job, err error)

	wg     sync.WaitGroup
	closed chan struct{}
	once   sync.Once
}

// JobQueueOption 用于配置 JobQueue。
type JobQueueOption func(*JobQueue)

// WithJobOnError 设置 Job 执行失败时的回调。
func WithJobOnError(fn func(job *Job, err error)) JobQueueOption {
	return func(q *JobQueue) {
		q.onError = fn
	}
}

// NewJobQueue 创建一个带 workerNum 个 worker、队列长度为 queueSize 的 JobQueue。
// workerNum 或 queueSize 小于等于 0 时会退化为 1。
func NewJobQueue(workerNum, queueSize int, opts ...JobQueueOption) *JobQueue {
	if workerNum <= 0 {
		workerNum = 1
	}
	if queueSize <= 0 {
		queueSize = 1
	}
	q := &JobQueue{
		jobs:    make(map[string]*Job),
		queue:   make(chan *jobEntry, queueSize),
		workers: workerNum,
		closed:  make(chan struct{}),
	}
	for _, opt := range opts {
		opt(q)
	}
	q.start()
	return q
}

func (q *JobQueue) start() {
	for i := 0; i < q.workers; i++ {
		q.wg.Add(1)
		go q.worker()
	}
}

// Submit 提交一个 Job，返回 Job ID。
// topic 用于按业务分类；meta 用于挂载轻量级元数据（例如 repo/branch）。
// 若队列已满，则会阻塞直到有空位或 ctx 被取消。
func (q *JobQueue) Submit(topic string, meta map[string]string, ctx context.Context, fn JobFunc) (string, error) {
	select {
	case <-q.closed:
		return "", context.Canceled
	default:
	}

	if ctx == nil {
		ctx = context.Background()
	}

	id := uuid.NewString()
	now := time.Now()

	m := meta
	if m == nil {
		m = make(map[string]string)
	}

	job := &Job{
		ID:         id,
		Topic:      topic,
		Status:     JobPending,
		Meta:       m,
		EnqueuedAt: now,
	}

	q.mu.Lock()
	q.jobs[id] = job
	q.mu.Unlock()

	entry := &jobEntry{
		job: job,
		ctx: ctx,
		fn:  fn,
	}

	select {
	case q.queue <- entry:
		return id, nil
	case <-ctx.Done():
		// 调用方上下文取消，标记为 canceled
		q.mu.Lock()
		if info, ok := q.jobs[id]; ok && info.Status == JobPending {
			info.Status = JobCanceled
			info.FinishedAt = time.Now()
			info.Error = ctx.Err().Error()
		}
		q.mu.Unlock()
		return "", ctx.Err()
	case <-q.closed:
		return "", context.Canceled
	}
}

func (q *JobQueue) worker() {
	defer q.wg.Done()

	for entry := range q.queue {
		q.runOne(entry)
	}
}

func (q *JobQueue) runOne(entry *jobEntry) {
	// 为 Job 创建一个可取消的子 context，便于在 Close 时整体取消。
	ctx, cancel := context.WithCancel(entry.ctx)
	defer cancel()

	q.mu.Lock()
	info, ok := q.jobs[entry.job.ID]
	if !ok {
		q.mu.Unlock()
		return
	}
	if info.Status != JobPending {
		q.mu.Unlock()
		return
	}
	info.Status = JobRunning
	info.StartedAt = time.Now()
	q.mu.Unlock()

	err := entry.fn(ctx, entry.job)

	q.mu.Lock()
	defer q.mu.Unlock()

	info.FinishedAt = time.Now()
	if err != nil {
		if ctx.Err() != nil && errors.Is(ctx.Err(), context.Canceled) {
			info.Status = JobCanceled
			info.Error = ctx.Err().Error()
		} else {
			info.Status = JobFailed
			info.Error = err.Error()
		}
		if q.onError != nil {
			// 回调本身不阻塞 JobQueue
			go q.onError(info, err)
		}
		return
	}

	if ctx.Err() != nil && errors.Is(ctx.Err(), context.Canceled) {
		info.Status = JobCanceled
		info.Error = ctx.Err().Error()
		return
	}

	info.Status = JobSucceeded
	info.Error = ""
}

// Jobs 返回当前所有 Job 的快照，用于列表观测。
func (q *JobQueue) Jobs() []Job {
	q.mu.RLock()
	defer q.mu.RUnlock()

	out := make([]Job, 0, len(q.jobs))
	for _, j := range q.jobs {
		cp := *j
		out = append(out, cp)
	}
	return out
}

// JobsByTopic 返回指定 topic 下的 Job 快照。
func (q *JobQueue) JobsByTopic(topic string) []Job {
	q.mu.RLock()
	defer q.mu.RUnlock()

	out := make([]Job, 0)
	for _, j := range q.jobs {
		if j.Topic == topic {
			cp := *j
			out = append(out, cp)
		}
	}
	return out
}

// Job 返回指定 Job 的快照。
func (q *JobQueue) Job(id string) (Job, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	j, ok := q.jobs[id]
	if !ok {
		return Job{}, false
	}
	cp := *j
	return cp, true
}

// Close 关闭 JobQueue，停止接收新 Job，并等待队列中的 Job 执行完成。
func (q *JobQueue) Close() {
	q.once.Do(func() {
		close(q.closed)
		close(q.queue)
		q.wg.Wait()
	})
}

