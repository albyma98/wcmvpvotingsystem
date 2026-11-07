package api

// Close should close everything opened in the lifecycle of the `_router`; for example, background goroutines.
func (rt *_router) Close() error {
	if rt.merchWorker != nil {
		rt.merchWorker.cancel()
		<-rt.merchWorker.done
	}
	if rt.votingNotifications != nil && rt.votingNotifications.cancel != nil {
		rt.votingNotifications.cancel()
		<-rt.votingNotifications.done
	}
	if rt.clubDomains != nil && rt.clubDomains.cancel != nil {
		rt.clubDomains.cancel()
		<-rt.clubDomains.done
	}
	return nil
}
