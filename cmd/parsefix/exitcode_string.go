// Code generated by "stringer -type=exitCode"; DO NOT EDIT.

package main

import "strconv"

const _exitCode_name = "fixedSomeExitfixedNoneExitnothingToFixExiterrorExit"

var _exitCode_index = [...]uint8{0, 13, 26, 42, 51}

func (i exitCode) String() string {
	if i < 0 || i >= exitCode(len(_exitCode_index)-1) {
		return "exitCode(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _exitCode_name[_exitCode_index[i]:_exitCode_index[i+1]]
}