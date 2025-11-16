package gpg

// This package will contain helper functions for GPG signing.
// It can either use a Go library like `ProtonMail/go-crypto` or
// shell out to the `gpg` command-line tool.

// SignFile signs a file with GPG.
func SignFile(filePath, keyEmail string) error {
	// TODO: Implement file signing logic.
	// This will involve:
	// 1. Finding the GPG key to use.
	// 2. Creating a detached signature for the file.
	return nil
}
