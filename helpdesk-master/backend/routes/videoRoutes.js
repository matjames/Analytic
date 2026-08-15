import express from "express";
import { getVideos, createVideo, updateVideo, deleteVideo } from "../controllers/videoController.js";
import Auth, { requireAdmin } from "../middleware/auth.js";

const router = express.Router();

router.get("/", Auth, getVideos);
import { upload } from "../controllers/videoController.js";

// Mutations are protected: unauthenticated or non-admin writes are DENIED.
router.post('/', Auth, requireAdmin, upload.single('video'), createVideo);
router.put("/:id", Auth, requireAdmin, updateVideo);
router.delete("/:id", Auth, requireAdmin, deleteVideo);

export default router;
