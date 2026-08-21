# Bug
Closing a review returns a closed object without its captured snapshot.

# Trigger
Close a seeded review round with recipients and inspect the returned value.

# Error
The test reports that the review did not close with a snapshot.
