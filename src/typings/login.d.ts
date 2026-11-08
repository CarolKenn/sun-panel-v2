declare namespace Login{

    interface LoginReqest{
        username:string 
        password:string
    }

	interface LoginResponse extends User.Info{
		token :string
	}

}