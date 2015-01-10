TODO
----

# In Progress

- dc
	- Needs refactoring
		- Set up a better interpreter structure so we aren't consuming
		  stack space so rapidly.
		- Either centralize number vs. command checking, or simply
		  store commands as big numbers & don't worry about the
		  difference.
			- Leaning towards second option.
		- Make 'l' and 's' registers separate from 'L' and 'S'
		  registers
		- Figure out ways to either simplify or eliminate need to
		  duplicate math.* functions for integers
		- Separate interpreter logic from parser logic
			- i.e. Put interpreter functions into functions instead
			  of inline commands
		- Find and eliminate code duplication where it makes sense.
	- Implement `Z`
	- Implement `?`
	- Decide on implementation or exclusion of `!`
		- Leaning towards No, after all, why does a desktop calculator
		  need the ability to execute shell commands? Seems like a
		  potential security vulnerability.
	- Make `q` and `Q` work with recursive nature of current interpreter
	- Implement `v`
	- Make this actually work with big integers
		- Likely just use the golang `math/big.Rat` type & calculate
		  the scale on demand
	- Implement `0x`, `0o`, and 0b to specify alternate number bases to
	  conform to current programming language standards.

# Still to implement

- Tools from plan9 Goblin requirements
	- ascii
	- awk
	- bc
	- cal
	- cat
	- cleanname
	- cmp
	- date
	- du
	- dd
	- diff
	- echo
	- ed
	- factor
	- fortune
	- fmt
	- freq
	- getflags
	- grep
	- hoc
	- join
	- look
	- ls
	- mk
	- mkdir
	- mtime
	- pbd
	- primes
	- rc
	- read
	- sam
	- sha1sum
	- sed
	- seq
	- sort
	- split
	- strings
	- tail
	- tee
	- test
	- touch
	- tr
	- troff
	- unicode
	- uniq
	- unutf
- Document where implemented tools are different from existing tools, and why.
	 
