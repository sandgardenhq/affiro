# asig

asig is the primary SDK, potentially public library for building affiro signatures.

## v0.0.2 ideation

- to save space, we could collape pauses; i.e. if byte[i] = 0, then byte[i+1] is the length of the pause in 4 second chunks; that'd let us track up to 17 minutes of sleep in two bytes; on the other hand if 4 second waits are terribly common it would make those waits 2 bytes instead of 1. Hunch: waits longer than 8 seconds are going to be more frequent, or just as frequent, as 4 second waits, so two bytes would be an improvement; q: what about waits longer than 17 minutes? a: we don't care about tracking longer waits 
- tracking other interesting actions (tab (for tab completion tooling), delete (to subtract from total characters input)) would be potentially valuable, but we should hypotheisze the algorithm we want that wants this information before adding it. Adding any more information will be tricky to do so without significantly expanding the space of the signature. 

(this was completed)