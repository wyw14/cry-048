# Bug
Concurrent edits using the same expected revision both commit successfully.

# Trigger
Run the targeted collaboration race test with two synchronized editors.

# Error
The test reports two successes and zero conflicts.
