import Video from "../models/videoModels.js";
import multer from "multer";

const storage = multer.diskStorage({
  destination: (req, file, cb) => {
    cb(null, "uploads/");
  },
  filename: (req, file, cb) => {
    cb(null, Date.now() + "-" + file.originalname);
  },
});

const upload = multer({
  storage: storage,
  limits: { fileSize: 1024 * 1024 * 500 }, // 500MB
});

export { upload };

// GET /api/videos
export const getVideos = async (req, res) => {
  try {
    const videos = await Video.findAll();
    // Map to add the url property
    const videosWithUrl = videos.map((v) => ({
      id: v.id,
      title: v.title,
      // In your getVideos controller
      url: v.video
        ? `${req.protocol}://${req.get("host")}/uploads/${v.video}`
        : null,
      // add other fields if needed
    }));
    res.json(videosWithUrl);
  } catch (err) {
    res.status(500).json({ message: "Server error", error: err.message });
  }
};
// PUT /api/videos/:id
export const updateVideo = async (req, res) => {
  const { title, url, thumbnail } = req.body;
  try {
    const video = await Video.findByPk(req.params.id);
    if (!video) {
      return res.status(404).json({ message: "Video not found" });
    }
    video.title = title || video.title;
    video.url = url || video.url;
    video.thumbnail = thumbnail || video.thumbnail;
    await video.save();
    res.json(video);
  } catch (err) {
    res.status(500).json({ message: "Server error", error: err.message });
  }
};

// DELETE /api/videos/:id
export const deleteVideo = async (req, res) => {
  try {
    const video = await Video.findByPk(req.params.id);
    if (!video) {
      return res.status(404).json({ message: "Video not found" });
    }
    await video.destroy();
    res.status(204).send();
  } catch (err) {
    res.status(500).json({ message: "Server error", error: err.message });
  }
};
// POST /api/videos
export const createVideo = async (req, res) => {
  const { title } = req.body;
  if (!title || !req.file) {
    return res.status(400).json({
      message: "All fields are required",
      missing: { title: !title, file: !req.file },
    });
  }

  try {
    const { filename } = req.file;
    const video = await Video.create({
      title,
      video: filename,
    });
    res.status(201).json(video);
  } catch (err) {
    console.error(err);
    res.status(500).json({ message: "Server error", error: err.message });
  }
};
