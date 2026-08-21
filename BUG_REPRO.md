# Bug
Canceled CSV exports still commit a downloadable artifact.

# Trigger
Cancel the request context before starting annotation export.

# Error
Export returns success instead of the cancellation error.
