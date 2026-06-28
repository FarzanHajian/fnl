/*
    Copyright (c) 2026 Farzan Hajian
    SPDX-License-Identifier: BSD-3-Clause

    The code calculates Pi using Gregory Series:

        Pi = 4 * (1 - 1/3 + 1/5 - 1/7 + 1/9 - 1/11 + ...)
*/

var MIN_TERM_COUNT:int = 1000
var term_count:int = 0
var invalid_input:bool = true
while invalid_input == true {
    print("Enter the number of terms (minimum " + to_str(MIN_TERM_COUNT) + ") to calculate Pi: ")
    var input_string:string = input()
    if is_int(input_string) {
        term_count = to_int(input_string)
        if term_count < MIN_TERM_COUNT {
            invalid_input = true
        }
        else {
            invalid_input = false
        }
    }
    else {
        invalid_input=true
    }
}

var pi:double = 0.0
var term_index:int = 1 
var sign:double = 1.0
var divisor:int = 1
while term_index <= term_count {
    pi = pi + sign * (1.0 / divisor)
    sign = -1.0 * sign
    divisor = divisor + 2
    term_index = term_index + 1
}

pi = pi * 4
println("Calculated value of Pi using " + to_str(term_count) + " terms is: " + to_str(pi))