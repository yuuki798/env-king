package webspider

type MirrorStatus struct {
	Address string
	Status  string
}

//type MyDebugger struct {
//	// Output is the log destination, anything can be used which implements them
//	// io.Writer interface. Leave it blank to use STDERR
//	Output io.Writer
//	// Prefix appears at the beginning of each generated log line
//	Prefix string
//	// Flag defines the logging properties.
//	Flag    int
//	logger  *log.Logger
//	counter int32
//	start   time.Time
//}
//
//// Init initializes the LogDebugger
//func (this *MyDebugger) Init() error {
//	this.counter = 0
//	this.start = time.Now()
//	if this.Output == nil {
//		this.Output = io.Discard
//	}
//	this.Output = nil // 直接静默
//	this.logger = log.New(this.Output, this.Prefix, this.Flag)
//	return nil
//}
//
//// Event receives Collector events and prints them to STDERR
//func (this *MyDebugger) Event(e *debug.Event) {
//	i := atomic.AddInt32(&this.counter, 1)
//	this.logger.Printf("[%06d] %d [%6d - %s] %q (%s)\n", i, e.CollectorID, e.RequestID, e.Type, e.Values, time.Since(this.start))
//}
