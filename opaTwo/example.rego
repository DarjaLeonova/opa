package example

default allow = false

allow {
    input.method = "GET"
    input.path = ["users", login]
	allowed[user]
    user.login = login
}

allowed[user] {
    user = data.users[_]
	user.login = input.customer.login
	user.password = input.customer.password
}
