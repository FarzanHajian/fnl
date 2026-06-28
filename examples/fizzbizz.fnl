/*
    Copyright (c) 2026 Farzan Hajian
    SPDX-License-Identifier: BSD-3-Clause

    FizzBuzz in FNL
*/

var i:int = 1
while i <= 100 {
    if i % 15 == 0 {
        println("FizzBuzz")
    } elseif i % 3 == 0 {
        println("Fizz")
    } elseif i % 5 == 0 {
        println("Buzz")
    } else {
        println(to_str(i))
    }
    i = i + 1
}