import React from 'react'

const StatsCard = ({ title, number, color }) => {
    return (
        // <div className="col-md-3 grid-margin" >
        //     <div className={`card ${color}`}>
        //         <div className="card-body">
        //             <div className="d-flex justify-content-between align-items-baseline">
        //                 <h6 className="card-title mb-0">{title}</h6>
        //                 <div className="dropdown mb-2">
        //                     <div className="dropdown-menu" aria-labelledby="dropdownMenuButton">
        //                         <a className="dropdown-item d-flex align-items-center" href="javascript:;"><svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" className="feather feather-eye icon-sm me-2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path><circle cx="12" cy="12" r="3"></circle></svg> <span className="">View</span></a>
        //                         <a className="dropdown-item d-flex align-items-center" href="javascript:;"><svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" className="feather feather-edit-2 icon-sm me-2"><path d="M17 3a2.828 2.828 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5L17 3z"></path></svg> <span className="">Edit</span></a>
        //                         <a className="dropdown-item d-flex align-items-center" href="javascript:;"><svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" className="feather feather-trash icon-sm me-2"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg> <span className="">Delete</span></a>
        //                         <a className="dropdown-item d-flex align-items-center" href="javascript:;"><svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" className="feather feather-printer icon-sm me-2"><polyline points="6 9 6 2 18 2 18 9"></polyline><path d="M6 18H4a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2h-2"></path><rect x="6" y="14" width="12" height="8"></rect></svg> <span className="">Print</span></a>
        //                         <a className="dropdown-item d-flex align-items-center" href="javascript:;"><svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" className="feather feather-download icon-sm me-2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="7 10 12 15 17 10"></polyline><line x1="12" y1="15" x2="12" y2="3"></line></svg> <span className="">Download</span></a>
        //                     </div>
        //                 </div>
        //             </div>
        //             <div className="row">
        //                 <div className="col-6 col-md-12 col-xl-5">
        //                     <div classNameName="d-flex flex-row">
        //                         <h3 className="mb-2">{number}</h3>
        //                         {/* <h3 className="mb-2">{number}</h3> */}
        //                     </div>
        //                 </div>
        //             </div>
        //         </div>
        //     </div>
        // </div>
        <div class="col-xl-3 col-sm-6 col-12 grid-margin">
            <div class={`card ${color}`}>
                <div class="card-content">
                    <div class="card-body">
                        <div class="media d-flex">
                            <div class="media-body text-left">
                                <h3 class="success">{number}</h3>
                                <span>{title}</span>
                            </div>
                            <div class="align-self-center">
                                {/* <i class="icon-user success font-large-2 float-right"></i> */}
                                {/* <img class="wd-30 ht-30 rounded-circle success font-large-2 float-right" src="images/user.svg" alt="profile" /> */}
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default StatsCard