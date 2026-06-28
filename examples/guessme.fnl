/*
    Copyright (c) 2026 Farzan Hajian
    SPDX-License-Identifier: BSD-3-Clause
*/

var upper:int = 0
var input_text:string = ""

print("Enter the upper limit for the game: ")
while upper <= 0 {
    var has_error:bool = false
    input_text = input()
    if is_int(input_text) {
        upper = to_int(input_text)
        if (upper < 1) {
            has_error = true
        }        
    } else {
        has_error = true
    }

    print("Please enter a positive integer:")
}

upper = to_int(input_text)

var secret:int = 1 + to_int(math_random() * to_double(upper))
var guessed:bool = false

println("Guess a number from 1 to " + to_str(upper))

var tries:int = 1
while guessed == false {
    print("Your guess: ")
    input_text = input()

    if is_int(input_text) {
        var guess:int = to_int(input_text)

        if guess < secret {
            println("Too low.")
            tries = tries + 1
        } elseif guess > secret {
            println("Too high.")
            tries = tries + 1
        } else {
            println("Correct! You guessed it after " + to_str(tries) + " tries.")
            guessed = true
        }
    } else {
        println("Please enter an integer.")
    }
}
