package main

import data.library

violation[msg] {
	m := "hello"
	other_method(m)
	library.hello(m)
	msg = m
}
