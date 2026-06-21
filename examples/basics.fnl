/*
    Copyright (c) 2026 Farzan Hajian
    SPDX-License-Identifier: BSD-3-Clause
*/

var name:string="FNL"
var x:int64=2
var y:int64=5
var d:double=3.5
var ok:bool=x<y
var power:double=x^y

println("hello from " + name)
println("x + y = " + to_str(x+y))
println("d + x = " + to_str(d+x))
println("x ^ y = " + to_str(power))
println("ok = " + to_str(ok))

if ok {
    println("if branch")
} else {
    println("else branch")
}

while x<5 {
    println("loop x = " + to_str(x))
    x=x+1
}

print("Enter an int64 value: ")
var user_input:string=input()

if is_int64(user_input) {
    var value:int64=to_int64(user_input)
    println("Parsed value: " + to_str(value))
} else {
    println("Invalid int64 value")
}

print("Enter a double value: ")
var double_input:string=input()

if is_double(double_input) {
    var double_value:double=to_double(double_input)
    println("Parsed double: " + to_str(double_value))
} else {
    println("Invalid double value")
}
