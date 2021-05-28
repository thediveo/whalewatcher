goversion = 1.16 1.15
phonies = help clean godoc reportcard

# A literal space; see: https://stackoverflow.com/a/9551487
space :=
space +=

# A comma followed by a space.
commasep := ,
commasep +=

comma-joined = $(subst $(space),$(commasep),$(strip $1))

.PHONY: $(phonies)

help:
	@printf "available targets: $(call comma-joined,$(phonies))\n"

clean:
	rm -f coverage.html coverage.out coverage.txt

godoc:
	@godoc -http=:6060

reportcard:
	@scripts/goreportcard
