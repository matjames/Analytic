import React from 'react'

const FNEmpty = ({open, title, title1, title2}) => {
    return (
        <div className="row">
            <div class="card bg-primary-subtle">
                <div class="row">
                    <div class="col">
                        <div class="text-primary p-3">
                            <h5 class="text-primary">{title}</h5>
                            <p>{title1}</p>
                        </div>
                    </div>
                    <div className="col">
                        <div class="float-end p-4">
                            <button class="btn btn-primary" type="button" onClick={open}>
                                <i class="bx bxs-cog align-middle me-1"></i> {title2}
                            </button>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default FNEmpty