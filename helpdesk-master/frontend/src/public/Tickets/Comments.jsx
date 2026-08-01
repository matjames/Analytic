import React, { useState, useEffect } from 'react';
import { toast } from "react-toastify";
import { Spinner } from "react-bootstrap";
import API from "../../helpers/api";

const Comments = ({ id }) => {
    const [comments, setComments] = useState([]);
    const [commentedby, setCommentedBy] = useState('');
    const [email, setEmail] = useState('');
    const [comment, setComment] = useState('');
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        fetchComments();
    }, []);

    const fetchComments = async () => {
        try {
            const res = await API.get(`/t/comments/ticket/${id}`);
            setComments(res.data.comments);
        } catch (error) {
            console.error('Error fetching comments:', error);
        }
    };

    const handleSubmit = async (e) => {
        e.preventDefault();
        setLoading(true);
        const newComment = { commentedby, email, comment, ticketId: id };

        try {
            await API.post(`/t/comments`, newComment);
            setCommentedBy('');
            setEmail('');
            setComment('');
            fetchComments();
            toast.success('Comment Added Successfully !!');
            setLoading(false);
        } catch (error) {
            console.error('Error adding comment:', error);
            toast.error('Error Adding Comment !!');
        }
    };

    return (
        <div className="row justify-content-center">
            <div className="col-xl-10">
                <div className="mt-3">
                    {comments && <h5 className="font-size-15"><i className="bx bx-comment-dots text-muted align-middle me-1"></i> Comments :</h5>}
                    <div>
                        {comments.map(comment => (
                            <div className="d-flex py-3 border-top" key={comment.id}>
                                <div className="flex-shrink-0 me-3">
                                    <div className="avatar-xs">
                                        <div className="avatar-title rounded-circle bg-light text-primary">
                                            <i className="bx bxs-user"></i>
                                        </div>
                                    </div>
                                </div>
                                <div className="flex-grow-1">
                                    <h5 className="font-size-14 mb-1">{comment.commentedby} <small className="text-muted float-end">{new Date(comment.createdAt).toLocaleString()}</small></h5>
                                    <p className="text-muted">{comment.comment}</p>
                                </div>
                            </div>
                        ))}
                    </div>
                </div>

                <div className="mt-4">
                    <h5 className="font-size-16 mb-3">Add a Comment</h5>
                    <form onSubmit={handleSubmit}>
                        <div className="row">
                            <div className="col-md-6">
                                <div className="mb-3">
                                    <label htmlFor="commentname-input" className="form-label">Name</label>
                                    <input
                                        type="text"
                                        className="form-control"
                                        id="commentname-input"
                                        placeholder="Enter name"
                                        value={commentedby}
                                        onChange={(e) => setCommentedBy(e.target.value)}
                                    />
                                </div>
                            </div>
                            <div className="col-md-6">
                                <div className="mb-3">
                                    <label htmlFor="commentemail-input" className="form-label">Email</label>
                                    <input
                                        type="email"
                                        className="form-control"
                                        id="commentemail-input"
                                        placeholder="Enter email"
                                        value={email}
                                        onChange={(e) => setEmail(e.target.value)}
                                    />
                                </div>
                            </div>
                        </div>

                        <div className="mb-3">
                            <label htmlFor="commentmessage-input" className="form-label">Message</label>
                            <textarea
                                className="form-control"
                                id="commentmessage-input"
                                placeholder="Your comment here..."
                                rows="3"
                                value={comment}
                                onChange={(e) => setComment(e.target.value)}
                            ></textarea>
                        </div>

                        <div className="text-end">
                            <button type="submit" className="btn btn-success w-sm">
                                {loading ? (
                                    <Spinner
                                        animation="border"
                                        variant="light"
                                        role="status"
                                        as="span"
                                    >
                                        <span className="visually-hidden">Loading...</span>
                                    </Spinner>
                                ) : (
                                    "Add Comment"
                                )}
                            </button>
                        </div>
                    </form>
                </div>
            </div>
        </div>
    );
};

export default Comments;