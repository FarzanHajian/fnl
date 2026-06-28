/*
    Copyright (c) 2026 Farzan Hajian
    SPDX-License-Identifier: BSD-3-Clause

    The code calculates square roots using the long division method
*/

var number:int = 0

print("Enter a positive integer to calculate the root: ")
var in:string = input()
var invalid_input:bool = false
if is_int(in) {
    number = to_int(in)
    if number <= 0 {
        invalid_input = true
    }
}
else {
    invalid_input=true
}

if invalid_input {
    println("Sorry! I need a positive integer.")
    exit(1)
}

var result:int = 0
var dividend:int = 0
var num:int = number
var pair_count:int = -1
var fractional_digits_count:int = 0

while true {
    var falldown:int = 0
    if pair_count == -1 {
        falldown = num
        pair_count = 1
        while falldown >= 100 {
            falldown = (falldown - falldown % 100) / 100
            pair_count = pair_count + 1
        }
    } else {
        falldown = num / (100 ^ (pair_count - 1))
    }
    dividend = dividend * 100 + falldown
    num = num - (falldown * (100 ^ (pair_count - 1)))

    var doubled_root:int = result * 2
    var x:int = 0
    while x < 10 {
        var temp:int = (doubled_root * 10 + x) * x
        if temp < dividend {
            if x == 9 {
                break
            }
            x = x + 1
        } elseif temp == dividend {
            break
        } else {
            x = x - 1
            break
        }
    }    
    dividend = dividend - ((doubled_root * 10 + x) * x)
    result = result * 10 + x

    pair_count = pair_count -1
    if pair_count == 0 {
        if dividend == 0 {
            /* we've reached to and end */
            break
        } elseif fractional_digits_count == 10 {
            /* enough number of digits after the decimal point is alreay calculated */
            break
        } else {
            /* calculating one digit after the decimal point */
            fractional_digits_count = fractional_digits_count + 1
            pair_count = 1
        }
    }
}

print("sqrt(")
print(to_str(number))
print(") = ")
if fractional_digits_count == 0 {
    println(to_str(result))
} else{
    var double_result:double = (result * 1.0) / (10 ^ fractional_digits_count)
    println(to_str(double_result))
}