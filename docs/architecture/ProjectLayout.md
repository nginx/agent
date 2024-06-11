# Overview

This is a basic layout for NGINX Agent application. This document focusing only on the general layout and not what you have inside. It's basic so that the information is quickly digestable. 

## Directories

/cmd
Main applications for this project.

The directory name for each application should match the name of the executable (e.g., /cmd/agent).

/internal
Private application and library code. This is the code will not be exported and available outside of this project.

/pkg
Library code that's ok to use by external applications (e.g., /pkg/files and /pkg/uuid). Other projects will import these libraries expecting them to work. The I'll take pkg over internal blog post by Travis Jeffery provides a good overview of the pkg and internal directories and when it might make sense to use them.

It's also a way to group Go code in one place when your root directory contains lots of non-Go components and directories making it easier to run various Go tools (as mentioned in these talks: Best Practices for Industrial Programming from GopherCon EU 2018, GopherCon 2018: Kat Zien - How Do You Structure Your Go Apps and GoLab 2018 - Massimiliano Pippi - Project layout patterns in Go).

See the /pkg directory if you want to see which popular Go repos use this project layout pattern. This is a common layout pattern, but it's not universally accepted and some in the Go community don't recommend it.

It's ok not to use it if your app project is really small and where an extra level of nesting doesn't add much value (unless you really want to :-)). Think about it when it's getting big enough and your root directory gets pretty busy (especially if you have a lot of non-Go app components).

The pkg directory origins: The old Go source code used to use pkg for its packages and then various Go projects in the community started copying the pattern (see this Brad Fitzpatrick's tweet for more context).

/vendor
Application dependencies (managed manually or by your favorite dependency management tool like the new built-in Go Modules feature). The go mod vendor command will create the /vendor directory for you. Note that you might need to add the -mod=vendor flag to your go build command if you are not using Go 1.14 where it's on by default.

Don't commit your application dependencies if you are building a library.

Note that since 1.13 Go also enabled the module proxy feature (using https://proxy.golang.org as their module proxy server by default). Read more about it here to see if it fits all of your requirements and constraints. If it does, then you won't need the vendor directory at all.

## Other Directories
