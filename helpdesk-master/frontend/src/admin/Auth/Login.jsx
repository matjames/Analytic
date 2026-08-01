/* eslint-disable jsx-a11y/anchor-is-valid */
import React, { useState } from "react";
import { useHistory } from 'react-router-dom';
import { toast } from "react-toastify";
import API from "../../helpers/api";
import LoadSpinner from "../../components/FNSpinner";
import logo from './uganda logo.jpeg'

const Login = () => {

    const [username, setUserName] = useState("");
    const [password, setPassword] = useState("");
    const [loading, setLoading] = useState(false);

    const history = useHistory();

    const handleSubmit = async (e) => {
        setLoading(true)
        e.preventDefault();
        try {
            const res = await API.post(`/users/login`, { username, password });
            console.log(res)

            if (res.data.status === 'success') {
                toast.success(`User Login Successful`);
                setLoading(false)
                history.push('/admin/dashboard')
                localStorage.setItem("token", res.data.accessToken);
                localStorage.setItem("user", JSON.stringify(res.data.user));
            }
        } catch (error) {
            setLoading(false);
            toast.error(`User Login Failed`);
        }
    };

    return (
        <div class="account-pages my-5">
            <div class="container">
                <div class="row justify-content-center">
                    <div class="col-md-8 col-lg-6 col-xl-5">
                        <div class="card overflow-hidden">
                            <div class="bg-dark">
                                <div class="row">
                                    <div class="col-12">
                                        <div class="text-white p-4">
                                            <h5 class="text-white">StatGate Operations Platform</h5>
                                        </div>
                                    </div>
                                </div>
                            </div>
                            <div class="card-body pt-0">
                                <div class="auth-logo">
                                    <a href="#" class="auth-logo-light">
                                        <div class="avatar-md profile-user-wid mb-4">
                                            <span class="avatar-title rounded-circle bg-light">
                                                <img src={logo} alt="" class="rounded-circle" height="34" />
                                            </span>
                                        </div>
                                    </a>

                                    <a href="#" class="auth-logo-dark">
                                        <div class="avatar-md profile-user-wid mb-4">
                                            <span class="avatar-title rounded-circle bg-light">
                                                <img src={logo} alt="" class="rounded-circle" height="60" />
                                            </span>
                                        </div>
                                    </a>
                                </div>
                                <div class="p-2">
                                    <form class="form-horizontal">

                                        <div class="mb-3">
                                            <label for="username" class="form-label">Username</label>
                                            <input type="text" class="form-control" id="username" placeholder="Enter username"
                                                value={username}
                                                onChange={(e) => setUserName(e.target.value)}
                                            />
                                        </div>

                                        <div class="mb-3">
                                            <label class="form-label">Password</label>
                                            <div class="input-group auth-pass-inputgroup">
                                                <input type="password" class="form-control" placeholder="Enter password" aria-label="Password" aria-describedby="password-addon"
                                                    value={password}
                                                    onChange={(e) => setPassword(e.target.value)}
                                                />
                                                <button class="btn btn-light " type="button" id="password-addon"><i class="mdi mdi-eye-outline"></i></button>
                                            </div>
                                        </div>

                                        <div class="form-check">
                                            <input class="form-check-input" type="checkbox" id="remember-check" />
                                            <label class="form-check-label" for="remember-check">
                                                Remember me
                                            </label>
                                        </div>

                                        <div class="mt-3 d-grid">
                                            <button class="btn btn-dark waves-effect waves-light" type="submit" onClick={handleSubmit}>
                                                {loading ? (
                                                    <LoadSpinner />
                                                ) : (
                                                    'Login'
                                                )}
                                            </button>
                                        </div>

                                        <div class="mt-4 text-center">
                                            <a href="#" class="text-muted"><i class="mdi mdi-lock me-1"></i> Forgot your password?</a>
                                        </div>
                                    </form>
                                </div>

                            </div>
                        </div>
                        <div class="mt-5 text-center">
                            <div>
                                <p>Don't have an account ? <a href="#" class="fw-medium text-primary"> Contact IT Admin </a> </p>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>

    );
};

export default Login;