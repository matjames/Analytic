/* eslint-disable*/
import React, { useState, useEffect } from "react";
import API from "../../helpers/api";

const formStyle = {
  display: "flex",
  flexDirection: "column",
  alignItems: "center",
  marginBottom: "2rem",
  padding: "1rem",
  border: "1px solid #eee",
  borderRadius: "8px",
  background: "#fafbfc",
  maxWidth: 400,
  margin: "0 auto 2rem auto",
};

const inputStyle = {
  marginBottom: "10px",
  padding: "8px",
  borderRadius: "4px",
  border: "1px solid #ccc",
  width: "100%",
  maxWidth: "350px",
};

const buttonStyle = {
  backgroundColor: "#007bff",
  color: "white",
  cursor: "pointer",
  border: "none",
  borderRadius: "4px",
  padding: "8px 16px",
  marginRight: "8px",
};

const videoListStyle = {
  
    display: "grid",
  gridTemplateColumns: "repeat(3, 1fr)",
  gap: "1.5rem",
  alignItems: "center",
};

const videoCardStyle = {
  width: "100%",
  maxWidth: 500,
  background: "#fff",
  border: "1px solid #eee",
  borderRadius: "8px",
  padding: "1rem",
  boxShadow: "0 2px 8px rgba(0,0,0,0.03)",
};

const VideoUpload = () => {
  const [title, setTitle] = useState("");
  const [video, setVideo] = useState(null);
  const [previewUrl, setPreviewUrl] = useState(null);
  const [videos, setVideos] = useState([]);

  useEffect(() => {
    fetchVideos();
  }, []);

  const fetchVideos = async () => {
    try {
      const res = await API.get("/videos");
      setVideos(res.data);
      console.log("Fetched videos:", res.data); // <-- Add this
    } catch (err) {
      alert("Failed to get videos");
    }
  };

  const handleUpload = async (e) => {
    e.preventDefault();
    if (!video) return;

    const formData = new FormData();
    formData.append("title", title);
    formData.append("video", video);

    try {
      const res = await API.post("/videos", formData, {
        headers: { "Content-Type": "multipart/form-data" },
      });
      setVideos([...videos, res.data]);
      setTitle("");
      setVideo(null);
      setPreviewUrl(null);
      window.location.reload(); // <-- Reload the page after upload
    } catch (err) {
      alert("Upload failed");
    }
  };

  const handleUpdate = async (id, oldTitle) => {
    const newTitle = prompt("Enter new title", oldTitle);
    if (!newTitle || newTitle === oldTitle) return;
    try {
      await API.put(`/videos/${id}`, { title: newTitle });
      setVideos(
        videos.map((v) => (v.id === id ? { ...v, title: newTitle } : v))
      );
    } catch (err) {
      alert("Update failed");
    }
  };

  const handleDelete = async (id) => {
    if (!window.confirm("Delete this video?")) return;
    try {
      await API.delete(`/videos/${id}`);
      setVideos(videos.filter((v) => v.id !== id));
    } catch (err) {
      alert("Delete failed");
    }
  };

  return (
    <div>
      <form onSubmit={handleUpload} style={formStyle}>
        <h3>Upload Video</h3>
        <input
          type="text"
          placeholder="Video Title"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          required
          style={inputStyle}
        />
        <input
          type="file"
          accept="video/*"
          onChange={(e) => {
            const file = e.target.files[0];
            setVideo(file);
            setPreviewUrl(file ? URL.createObjectURL(file) : null);
          }}
          required
          style={inputStyle}
        />
        {previewUrl && (
          <video
            src={previewUrl}
            width="100%"
            height="180"
            style={{
              objectFit: "cover",
              borderRadius: "8px",
              marginBottom: "10px",
            }}
            controls
          />
        )}
        <button type="submit" style={buttonStyle}>
          Upload Video
        </button>
      </form>

      <h3 style={{ textAlign: "center" }}>Video List</h3>
      <div style={videoListStyle}>
        {videos.map((video) => (
          <div key={video.id} style={videoCardStyle}>
            <h5>{video.title}</h5>
            <video
              src={video.url}
              width="100%"
              height="220"
              style={{ objectFit: "cover", borderRadius: "8px" }}
              controls
            />
            <div style={{ marginTop: "10px" }}>
              <button
                style={buttonStyle}
                onClick={() => handleUpdate(video.id, video.title)}
              >
                Update Title
              </button>
              <button
                style={{ ...buttonStyle, backgroundColor: "#dc3545" }}
                onClick={() => handleDelete(video.id)}
              >
                Delete
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

export default VideoUpload;
