import express from "express";
import { getVideos, createVideo, updateVideo, deleteVideo } from "../controllers/videoController.js";

const router = express.Router();

router.get("/", getVideos);
import { upload } from "../controllers/videoController.js";

router.post('/', upload.single('video'), createVideo);
router.put("/:id", /* authMiddleware, */ updateVideo);
router.delete("/:id", /* authMiddleware, */ deleteVideo);

export default router;
