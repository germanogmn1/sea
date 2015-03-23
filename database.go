package main

import "strings"

var buildList = []*Build{
	&Build{
		Rev:    "134d74025fbbbbcac149f206d4157890e145e8c3",
		State:  BUILD_WAITING,
		Path:   ".",
		Output: NewEmptyOutputBuffer(),
	},
	&Build{
		Rev:    "bbdc1e3744f128dfa744ab5bed520c0e5ab2e116",
		State:  BUILD_SUCCESS,
		Path:   ".",
		Output: NewFilledOutputBuffer([]byte("success")),
	},
	&Build{
		Rev:    "c21e9b8ff5f55ceeacffeadfd6d5ca4fce8dc6a7",
		State:  BUILD_FAILED,
		Path:   ".",
		Output: NewFilledOutputBuffer([]byte("fail")),
	},
}

func AddBuild(b *Build) {
	buildList = append(buildList, b)
}

func FindBuild(rev string) *Build {
	for i := range buildList {
		if strings.HasPrefix(buildList[i].Rev, rev) {
			return buildList[i]
		}
	}
	return nil
}
