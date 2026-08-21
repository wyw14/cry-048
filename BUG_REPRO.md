# Bug
Project member email uniqueness is case-sensitive.

# Trigger
Invite the same normalized email twice with different casing.

# Error
The second invitation succeeds instead of returning already-exists.
