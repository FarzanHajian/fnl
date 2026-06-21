/*
    Copyright (c) 2026 Farzan Hajian
    SPDX-License-Identifier: BSD-3-Clause

    Fibonacci serie generator written entirely in FNL
*/

var total_terms:int64 = 20
var first:int64 = 0
var second:int64 = 1

println("Here are the first " + to_str(total_terms) + " terms of the Fibonacci Serie:")

var current_term:int64 = 1
print (to_str(first))

if total_terms > 1 {
    current_term = 2
    print (", ")
    print (to_str(second))
}

while current_term < total_terms {
    current_term = current_term + 1
    print (", ")
    second = first + second
    first = second - first
    print (to_str(second))
}

println ("")
