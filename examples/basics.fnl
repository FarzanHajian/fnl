/*
    Copyright (c) 2026 Farzan Hajian
    SPDX-License-Identifier: BSD-3-Clause
*/

var name:string="FNL"
var myname:string="فرزان"
var x:int=2
var y:int=5
var d:double=3.5
var ok:bool=x<y
var power:double=x^y

println("hello from " + name)
println("my name is " + myname)
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

print("Enter an int value: ")
var user_input:string=input()

if is_int(user_input) {
    var value:int=to_int(user_input)
    println("Parsed value: " + to_str(value))
} else {
    println("Invalid int value")
}

print("Enter a double value: ")
var double_input:string=input()

if is_double(double_input) {
    var double_value:double=to_double(double_input)
    println("Parsed double: " + to_str(double_value))
} else {
    println("Invalid double value")
}
