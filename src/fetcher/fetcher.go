package fetcher

type EmailFetcher interface {
	// Connects to the server and authenticates
	Connect() error
	// Returns a slice of all unique IDs currently in the INBOX
	GetUIDs() ([]string, error)
	// Downloads only the headers for a specific UID
	DownloadEmailHeaders(uid string) ([]byte, error)
	// Downloads the raw RFC 2822 bytes for a specific UID
	DownloadEmail(uid string) ([]byte, error)
	// Closes the connection cleanly
	Close() error
}
