# Bug
Notification recipient planning mutates the caller-owned slice.

# Trigger
Pass an unsorted recipient list with duplicates and compare it after planning.

# Error
The input order and values are changed in place.
