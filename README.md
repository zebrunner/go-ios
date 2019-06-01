[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![CircleCI](https://circleci.com/gh/danielpaulus/go-ios.svg?style=svg)](https://circleci.com/gh/danielpaulus/go-ios)
[![codecov](https://codecov.io/gh/danielpaulus/go-ios/branch/master/graph/badge.svg)](https://codecov.io/gh/danielpaulus/go-ios)
[![Go Report](https://goreportcard.com/badge/github.com/danielpaulus/go-ios)](https://goreportcard.com/report/github.com/danielpaulus/go-ios)
# go-ios-old
This is how the project started out. I needed something to learn go with :-)
It has no tests as I was not going to release it at first. Now I am keeping this branch for copy pasting
until every functionality has been migrated to master. 

```
iOS client v 0.01

Usage:
ios list [--details]
ios info [options]
ios syslog [options]
ios screenshot [options]
ios devicename [options] 
ios date [options]
ios diagnostics list [options]
ios pair [options]
ios forward [options] <hostPort> <targetPort>
ios -h | --help
ios --version

Options:
-h --help     Show this screen.
--version     Show version.
-u=<udid>, --udid     UDID of the device.
-o=<filepath>, --output

```
