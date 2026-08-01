import React, { useState } from "react";
import { toast } from "react-toastify";
import API from "../../helpers/api";
import FNSpinner from "../../components/FNSpinner";
import defaultAvatar from "../../components/Layout/admin/user.jpg";

const AddAgent = ({ close, refresh }) => {

    const [loading, setLoading] = useState(false);
    const [formData, setFormData] = useState({ phoneNo: "", username: "", system: "", email: "", role: "", 
    password:"", firstname: "", lastname: ""});
    const [profilePicture, setProfilePicture] = useState(null);
    const [previewUrl, setPreviewUrl] = useState(defaultAvatar);

    const handleFileChange = (event) => {
        const file = event.target.files[0];
        if (!file) return;
        setProfilePicture(file);
        const reader = new FileReader();
        reader.onload = () => setPreviewUrl(reader.result);
        reader.readAsDataURL(file);
    };

    const handleSubmit = async (e) => {
        e.preventDefault();
        setLoading(true);

        try {
            const payload = new FormData();
            Object.entries(formData).forEach(([key, value]) => payload.append(key, value));
            if (profilePicture) payload.append("profilePicture", profilePicture);

            const response = await API.post("/users/register", payload, {
                headers: { "Content-Type": "multipart/form-data" },
            });
            console.log(response)
            setLoading(false);
            close();
            refresh();
            toast.success(`Agent Has Been Added Successfully`);
        } catch (error) {
            console.log("error", error);
            setLoading(false);
            toast.error("Error Encountered while Adding Agent");
        }
    };

    return (
        <div class="card custom-card">
            <div class="card-body">
                <section id="kyc-verify-wizard-p-0" role="tabpanel" aria-labelledby="kyc-verify-wizard-h-0" class="body current" aria-hidden="false">
                    <div class="row">
                        <div class="col-lg-6">
                            <div class="mb-3">
                                <label for="kycselectcity-input" class="form-label">Username</label>
                                <input type="text" class="form-control" placeholder="Enter Username"
                                    value={formData.username}
                                    onChange={(e) =>
                                        setFormData({ ...formData, username: e.target.value })
                                    } />
                            </div>
                        </div>
                        <div class="col-lg-6">
                            <div class="mb-3">
                                <label for="kycfirstname-input" class="form-label">Password</label>
                                <input type="text" class="form-control" placeholder="Enter Password"
                                    value={formData.password}
                                    onChange={(e) =>
                                        setFormData({ ...formData, password: e.target.value })
                                    } />
                            </div>
                        </div>
                        <div class="col-lg-6">
                            <div class="mb-3">
                                <label for="kycselectcity-input" class="form-label">First Name</label>
                                <input type="text" class="form-control" placeholder="Enter First Name"
                                    value={formData.firstname}
                                    onChange={(e) =>
                                        setFormData({ ...formData, firstname: e.target.value })
                                    } />
                            </div>
                        </div>
                        <div class="col-lg-6">
                            <div class="mb-3">
                                <label for="kycfirstname-input" class="form-label">Last Name</label>
                                <input type="text" class="form-control" placeholder="Enter Last Name"
                                    value={formData.lastname}
                                    onChange={(e) =>
                                        setFormData({ ...formData, lastname: e.target.value })
                                    } />
                            </div>
                        </div>
                        <div class="col-lg-6">
                            <div class="mb-3">
                                <label for="kycselectcity-input" class="form-label">Email Address</label>
                                <input type="text" class="form-control" placeholder="Enter Email address"
                                    value={formData.email}
                                    onChange={(e) =>
                                        setFormData({ ...formData, email: e.target.value })
                                    } />
                            </div>
                        </div>
                        <div class="col-lg-6">
                            <div class="mb-3">
                                <label for="kycselectcity-input" class="form-label">Phone Number</label>
                                <input type="text" class="form-control" placeholder="Enter Phone Number"
                                    value={formData.phoneNo}
                                    onChange={(e) =>
                                        setFormData({ ...formData, phoneNo: e.target.value })
                                    } />
                            </div>
                        </div>
                        <div class="col-lg-6">
                            <div class="mb-3">
                                <label for="kycfirstname-input" class="form-label">Operations Platform</label>
                                <select class="form-select"
                                    value={formData.system}
                                    onChange={(e) =>
                                        setFormData({ ...formData, system: e.target.value })
                                    }>
                                    <option>Select Platform</option>
                                    <option value="eAFYA">eAFYA</option>
                                    <option value="Clinic Master">Clinic Master</option>
                                    <option value="echis">echis</option>
                                    <option value="DHIS2">DHIS2</option>
                                </select>
                            </div>
                        </div>
                        <div class="col-lg-6">
                            <div class="mb-3">
                                <label for="kycfirstname-input" class="form-label">User Role</label>
                                <select class="form-select"
                                    value={formData.role}
                                    onChange={(e) =>
                                        setFormData({ ...formData, role: e.target.value })
                                    }>
                                    <option>Select User Role</option>
                                    <option value="agent">Agent</option>
                                    <option value="admin">Admin</option>
                                </select>
                            </div>
                        </div>
                        <div class="col-lg-12">
                            <div class="mb-3">
                                <label class="form-label">Profile Picture</label>
                                <img src={previewUrl} alt="Profile preview" className="avatar-preview" />
                                <input type="file" class="form-control" accept="image/*" onChange={handleFileChange} />
                            </div>
                        </div>
                        {/* <div class="col-lg-6">
                            <div class="mb-3">
                                <label for="kycfirstname-input" class="form-label">Employment Status</label>
                                <select class="form-select"
                                    value={formData.employment}
                                    onChange={(e) =>
                                        setFormData({ ...formData, employment: e.target.value })
                                    }>
                                    <option>Select Employment Status</option>
                                    <option value="Permanet">Permanet</option>
                                    <option value="Contract">Contract</option>
                                </select>
                            </div>
                        </div>
                        <div class="col-lg-6">
                            <div class="mb-3">
                                <label for="kycfirstname-input" class="form-label">Employment Status</label>
                                <select class="form-select"
                                    value={formData.employment}
                                    onChange={(e) =>
                                        setFormData({ ...formData, employment: e.target.value })
                                    }>
                                    <option>Select Employment Status</option>
                                    <option value="Permanet">Permanet</option>
                                    <option value="Contract">Contract</option>
                                </select>
                            </div>
                        </div> */}
                    </div>
                    <div className="actions clearfix">
                        <button className="btn btn-primary waves-effect waves-light" onClick={handleSubmit} role="menuitem" style={{ cursor: 'pointer' }}>
                            {loading ? <FNSpinner /> : "Add Operations Agent"}
                        </button>
                    </div>
                </section>
            </div>
        </div>
    )
}

export default AddAgent