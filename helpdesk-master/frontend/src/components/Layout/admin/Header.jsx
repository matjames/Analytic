/* eslint-disable jsx-a11y/anchor-is-valid */
import React, { useState, useRef, useEffect } from 'react'
import { Link, useHistory } from 'react-router-dom'
import pic from './user.jpg'

const launcherApps = [
    { name: 'Dashboard', url: 'http://localhost:5000', icon: 'bx bx-home', external: true },
    { name: 'Dataset Catalog', url: 'http://localhost:5000/datasets', icon: 'bx bx-table', external: true },
    { name: 'Notebook', url: 'http://localhost:5000/notebook', icon: 'bx bx-book-open', external: true },
    { name: 'Semantic Registry', url: 'http://localhost:5000/semantic', icon: 'bx bx-brain', external: true },
    { name: 'ABAC Security', url: 'http://localhost:5000/abac', icon: 'bx bx-shield', external: true },
    { name: 'Executive Centre', url: 'http://localhost:5000/executive', icon: 'bx bx-bank', external: true },
    { name: 'System Launcher', url: 'http://localhost:3002', icon: 'bi bi-grid-3x3-gap-fill', external: true },
    { name: 'Register Portal', url: 'http://localhost:3000', icon: 'bx bx-edit', external: true },
]

const Header = () => {

    const history = useHistory();
    const [showLauncher, setShowLauncher] = useState(false)
    const launcherRef = useRef(null)
    const user = JSON.parse(localStorage.getItem("user") || "null")
    const avatarSrc = user?.profilePicture || pic

    const logout = () => {
        try {
            localStorage.removeItem("token");
            localStorage.removeItem("user");
            history.push('/')
        } catch (error) {
            console.log(error)
        }
    };

    useEffect(() => {
        const handleClickOutside = (event) => {
            if (launcherRef.current && !launcherRef.current.contains(event.target)) {
                setShowLauncher(false)
            }
        }
        document.addEventListener('mousedown', handleClickOutside)
        return () => document.removeEventListener('mousedown', handleClickOutside)
    }, [])

    return (
        <header id="page-topbar">
            <div class="navbar-header">
                <div class="d-flex align-items-center">
                <a class="navbar-logo" href="#">
                        <img src="/logo-new.png" alt="" height="50" />
                    </a>

                    <button type="button" class="btn btn-sm mx-5 font-size-16 d-lg-none header-item waves-effect waves-light" data-bs-toggle="collapse" data-bs-target="#topnav-menu-content">
                        <i class="fa fa-fw fa-bars"></i>
                    </button>
                    <p className='text-white' style={{fontSize: '18px'}}>StatGate Operations Command Center</p>
                </div>

                <div class="d-flex">
                    <div class="launcher-dropdown d-inline-block me-2" ref={launcherRef}>
                        <button type="button" class="btn header-item noti-icon waves-effect launcher-toggle"
                            aria-label="Open application launcher"
                            onClick={() => setShowLauncher(!showLauncher)}>
                            <i class="bi bi-grid-3x3-gap-fill"></i>
                        </button>
                        {showLauncher && (
                            <div class="app-launcher-menu launcher-menu p-3" style={{position: 'absolute', right: '0', top: 'calc(100% + 10px)', minWidth: '280px', zIndex: 2000}}>
                                {launcherApps.map((app) => (
                                    <a
                                        key={app.name}
                                        href={app.url}
                                        class="launcher-card d-flex align-items-center gap-2 mb-2"
                                        style={{padding: '10px 12px', borderRadius: '10px', background: '#f8fbff', color: '#12263f', textDecoration: 'none', border: '1px solid rgba(22,92,146,0.12)'}}
                                        onClick={() => setShowLauncher(false)}
                                    >
                                        <i class={app.icon} style={{fontSize: '1rem'}}></i>
                                        <span>{app.name}</span>
                                    </a>
                                ))}
                            </div>
                        )}
                    </div>

                    <div class="dropdown d-inline-block d-lg-none ms-2">
                        <button type="button" class="btn header-item noti-icon waves-effect" id="page-header-search-dropdown"
                            data-bs-toggle="dropdown" aria-haspopup="true" aria-expanded="false">
                            <i class="mdi mdi-magnify"></i>
                        </button>
                        <div class="dropdown-menu dropdown-menu-lg dropdown-menu-end p-0"
                            aria-labelledby="page-header-search-dropdown">

                            <form class="p-3">
                                <div class="form-group m-0">
                                    <div class="input-group">
                                        <input type="text" class="form-control" placeholder="Search ..." aria-label="Search input" />

                                        <button class="btn btn-primary" type="submit"><i class="mdi mdi-magnify"></i></button>s
                                    </div>
                                </div>
                            </form>
                        </div>
                    </div>

                    <div class="dropdown d-none d-lg-inline-block ms-1">
                        <button type="button" class="btn header-item noti-icon waves-effect" data-bs-toggle="fullscreen">
                            <i class="bx bx-fullscreen"></i>
                        </button>
                    </div>

                    {/* Removed old standalone launcher button; using the app launcher dropdown above */}

                    <div class="dropdown d-inline-block">
                        <button type="button" class="btn header-item noti-icon waves-effect" id="page-header-notifications-dropdown"
                            data-bs-toggle="dropdown" aria-haspopup="true" aria-expanded="false">
                            <i class="bx bx-bell bx-tada"></i>
                            <span class="badge bg-danger rounded-pill">3</span>
                        </button>
                        <div class="dropdown-menu dropdown-menu-lg dropdown-menu-end p-0"
                            aria-labelledby="page-header-notifications-dropdown">
                            <div class="p-3">
                                <div class="row align-items-center">
                                    <div class="col">
                                        <h6 class="m-0" key="t-notifications"> Notifications </h6>
                                    </div>
                                    <div class="col-auto">
                                        <a href="#!" class="small" key="t-view-all"> View All</a>
                                    </div>
                                </div>
                            </div>
                            <div data-simplebar style={{maxHeight: '230px'}}>
                                <a href="javascript: void(0);" class="text-reset notification-item">
                                    <div class="d-flex">
                                        <div class="avatar-xs me-3">
                                            <span class="avatar-title bg-primary rounded-circle font-size-16">
                                                <i class="bx bx-cart"></i>
                                            </span>
                                        </div>
                                        <div class="flex-grow-1">
                                            <h6 class="mb-1" key="t-your-order">Your order is placed</h6>
                                            <div class="font-size-12 text-muted">
                                                <p class="mb-1" key="t-grammer">If several languages coalesce the grammar</p>
                                                <p class="mb-0"><i class="mdi mdi-clock-outline"></i> <span key="t-min-ago">3 min ago</span></p>
                                            </div>
                                        </div>
                                    </div>
                                </a>

                            </div>
                            <div class="p-2 border-top d-grid">
                                <a class="btn btn-sm btn-link font-size-14 text-center" href="javascript:void(0)">
                                    <i class="mdi mdi-arrow-right-circle me-1"></i> <span key="t-view-more">View More..</span>
                                </a>
                            </div>
                        </div>
                    </div>

                    <div class="dropdown d-inline-block">
                        <button type="button" class="btn header-item waves-effect" id="page-header-user-dropdown"
                            data-bs-toggle="dropdown" aria-haspopup="true" aria-expanded="false">
                            <img class="rounded-circle header-profile-user" src={avatarSrc}
                                alt="Header Avatar" />
                            <span class="d-none d-xl-inline-block ms-1" key="t-henry">{user && user.username}</span>
                            <i class="mdi mdi-chevron-down d-none d-xl-inline-block"></i>
                        </button>
                        <div class="dropdown-menu dropdown-menu-end">
                            <a class="dropdown-item" href="#"><i class="bx bx-user font-size-16 align-middle me-1"></i> <span key="t-profile">Profile</span></a>
                            <a class="dropdown-item d-block" href="#"><span class="badge bg-success float-end">11</span><i class="bx bx-wrench font-size-16 align-middle me-1"></i> <span key="t-settings">Settings</span></a>
                            <a class="dropdown-item" href="#"><i class="bx bx-lock-open font-size-16 align-middle me-1"></i> <span key="t-lock-screen">Lock screen</span></a>
                            <div class="dropdown-divider"></div>
                            <a class="dropdown-item text-danger" onClick={logout} style={{pointer:'cursor'}}><i class="bx bx-power-off font-size-16 align-middle me-1 text-danger"></i> <span key="t-logout">Logout</span></a>
                        </div>
                    </div>
                </div>
            </div>
        </header>
    )
}

export default Header