import React from 'react'

const AddUser = ({ setOpen }) => {

    const handleSubmit = (e) => {
        e.preventDefault();
        console.log("Submitted Added ===>")
    }

    return (
        <div className="add">
            <div className="modal">
                <span className='close' onClick={() => setOpen(false)}>X</span>
                <h3>Add User</h3>
                <form onSubmit={handleSubmit}>
                    <div className="item">
                        <label htmlFor="">Add User</label>
                        <input type="text" placeholder='Add New User' />
                    </div>
                    <div className="item">
                        <label htmlFor="">Add User</label>
                        <input type="text" placeholder='Add New User' />
                    </div>
                    <div className="item">
                        <label htmlFor="">Add User</label>
                        <input type="text" placeholder='Add New User' />
                    </div>
                    <div className="item">
                        <label htmlFor="">Add User</label>
                        <input type="text" placeholder='Add New User' />
                    </div>
                </form>
                <button>Submit</button>
            </div>
        </div>
    )
}

export default AddUser