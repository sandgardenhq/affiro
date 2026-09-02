# To recompile

go build --buildmode=c-shared -o keylog.dll

## TODO

- The terminal where the program runs still crashes a while after the test is run. But no fatal OS level cascading crashes remain.
- We aren't properly deduping repeat key presses