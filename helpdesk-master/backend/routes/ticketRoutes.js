import express from "express";
import multer from "multer";
import xlsx from "xlsx";
import User from "../models/userModel.js";
import transporter from "../config/email.js";
import TicketModel from "../models/ticketModel.js";
import Auth from "../middleware/auth.js";

const router = express.Router();
const upload = multer({ dest: "uploads/" });

router.post("/", upload.single("image"), async (req, res) => {
  try {
    let imageUrl = null;
    if (req.file && req.file.filename) {
      imageUrl = `/uploads/${req.file.filename}`;
    }

    const ticketData = {
      ...req.body,
      ...(imageUrl && { imageUrl }),
    };

    const ticket = await TicketModel.create(ticketData);
    const maillist =
      "frank.mwesigwa1@gmail.com, inassazi@gmail.com, mkussipa@gmail.com";

    const mailOptions = {
      from: "frank.mwesigwa1@gmail.com",
      to: maillist,
      subject: "StatGate Operations - New Case Created",
      text: `A New Ticket with Ticket Number: ${ticket.id} has been created. - ${ticket.description}`,
    };

    transporter.sendMail(mailOptions, (error, info) => {
      if (error) {
        console.log(error);
        return res.status(500).send("Error sending email");
      } else {
        console.log("Email sent: " + info.response);
        return res.status(200).send("Comment added and email sent");
      }
    });

    res.status(201).json({
      status: "success",
      ticket,
    });
  } catch (error) {
    res.status(500).json({
      status: "error",
      message: error.message,
    });
  }
});

router.post("/upload", upload.single("file"), async (req, res) => {
  try {
    const filePath = req.file.path;
    const workbook = xlsx.readFile(filePath);
    const sheetName = workbook.SheetNames[0];
    const sheet = workbook.Sheets[sheetName];
    const data = xlsx.utils.sheet_to_json(sheet);

    await TicketModel.bulkCreate(
      data.map((row) => ({
        reportedby: row.reportedby,
        priority: row.priority,
        level: row.level,
        facility: row.facility,
        category: row.category,
        module: row.module,
        emrtype: row.emrtype,
        phoneno: row.phoneno,
        description: row.description,
        agentId: row.agentId,
        image: row.image,
        system: row.system,
        dhis2instance: row.dhis2instance,
        dhis2module: row.dhis2module,
      }))
    );

    res.status(200).send("File uploaded and data saved");
  } catch (error) {
    console.error("Error uploading file:", error);
    res.status(500).send("Error uploading file");
  }
});

router.get("/", async (req, res) => {
  try {
    const page = parseInt(req.query.page) || 1;
    const limit = parseInt(req.query.limit) || 10;
    const skip = (page - 1) * limit;

    const filters = {
      ...(req.query.category && { category: req.query.category }),
      ...(req.query.status && { status: req.query.status }),
      ...(req.query.system && { system: req.query.system }),
      ...(req.query.level && { level: req.query.level }),
      ...(req.query.facility && { facility: req.query.facility }),
    };

    const hasFilters = Object.keys(filters).length > 0;
    const whereCondition = hasFilters ? filters : {};

    const totalRecords = await TicketModel.count({ where: whereCondition });
    const totalPages = Math.ceil(totalRecords / limit);

    const tickets = await TicketModel.findAll({
      where: whereCondition,
      limit,
      offset: skip,
      include: [{ model: User }],
    });

    res.status(200).json({
      status: "success",
      results: tickets.length,
      tickets,
      totalRecords,
      totalPages,
      currentPage: page,
    });
  } catch (error) {
    res.status(500).json({
      status: "error",
      message: error.message,
    });
  }
});

router.get("/count", async (req, res) => {
  try {
    const tickets = await TicketModel.findAll();

    let open = 0;
    let closed = 0;
    let inprogress = 0;
    let overdue = 0;

    tickets.forEach((ticket) => {
      switch (ticket.status) {
        case "inprogress":
          inprogress++;
          break;
        case "closed":
          closed++;
          break;
        case "open":
          open++;
          break;
        case "overdue":
          overdue++;
          break;
        default:
          break;
      }
    });

    res.status(200).json({
      status: "success",
      results: tickets.length,
      tickets,
      statusCounts: {
        open: open,
        closed: closed,
        inprogress: inprogress,
        overdue: overdue,
      },
    });
  } catch (error) {
    res.status(500).json({
      status: "error",
      message: error.message,
    });
  }
});

router.patch("/:id", async (req, res) => {
  try {
    const result = await TicketModel.update(
      { ...req.body, updatedAt: Date.now() },
      {
        where: {
          id: req.params.id,
        },
      }
    );

    if (result[0] === 0) {
      return res.status(404).json({
        status: "fail",
        message: "Ticket with that ID not found",
      });
    }

    const ticket = await TicketModel.findByPk(req.params.id);

    const maillist =
      "frank.mwesigwa1@gmail.com, inassazi@gmail.com, mkussipa@gmail.com";

    const mailOptions = {
      from: "frank.mwesigwa1@gmail.com",
      to: maillist,
      subject: "StatGate Operations - Case Updated",
      text: `The ticket with Ticket Number: ${ticket.id} has been updated. - ${ticket.description}`,
    };

    transporter.sendMail(mailOptions, (error, info) => {
      if (error) {
        console.log(error);
        return res.status(500).send("Error sending email");
      } else {
        console.log("Email sent: " + info.response);
        return res.status(200).json({
          status: "success",
          ticket,
          message: "Ticket updated and email sent",
        });
      }
    });
  } catch (error) {
    res.status(500).json({
      status: "error",
      message: error.message,
    });
  }
});

router.get("/:id", async (req, res) => {
  try {
    const ticket = await TicketModel.findByPk(req.params.id, {
      include: [{ model: User }],
    });

    if (!ticket) {
      return res.status(404).json({
        status: "fail",
        message: "Ticket with that ID not found",
      });
    }

    res.status(200).json({
      status: "success",
      ticket,
    });
  } catch (error) {
    res.status(500).json({
      status: "error",
      message: error.message,
    });
  }
});
/**
 *@swagger
 * /t/tickets/{id}:
 *   delete:
 *     summary: Delete a ticket
 *     tags:
 *       - Tickets
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *         description: ID of the ticket to delete
 * responses:
 *   204:
 *     description: Ticket deleted successfully
 *   404:
 *     description: Ticket not found
 *
 */
router.delete("/:id", async (req, res) => {
  try {
    const result = await TicketModel.destroy({
      where: { id: req.params.id },
      force: true,
    });

    if (result === 0) {
      return res.status(404).json({
        status: "fail",
        message: "Ticket with that ID not found",
      });
    }

    res.status(204).json();
  } catch (error) {
    res.status(500).json({
      status: "error",
      message: error.message,
    });
  }
});
/**
 * @swagger
 * /t/tickets/{id}/assign:
 *   post:
 *     summary: Assign a ticket to a user/agent
 *     tags:
 *       - Tickets
 *     parameters:
 *       - in: path
 *         name: id
 *         required: true
 *         schema:
 *           type: string
 *         description: ID of the ticket to assign
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             required:
 *               - agentId
 *             properties:
 *               agentId:
 *                 type: integer
 *                 description: ID of the user/agent to assign the ticket to
 *     responses:
 *       201:
 *         description: Ticket assigned successfully
 *       404:
 *         description: Ticket or user not found
 *       500:
 *         description: Failed to assign ticket
 */
router.post("/:id/assign", Auth, async (req, res) => {
  try {
    const { id: ticketId } = req.params;
    const { agentId } = req.body;

    if (!agentId) {
      return res.status(400).json({ error: "Agent ID is required" });
    }

    const ticket = await TicketModel.findByPk(ticketId);
    if (!ticket) {
      return res.status(404).json({ error: "Ticket not found" });
    }

    const agent = await User.findByPk(agentId);
    if (!agent) {
      return res.status(404).json({ error: "Agent not found" });
    }

    // Assign the agent to the ticket
    ticket.assignedTo = agentId;
    await ticket.save();

    return res.status(200).json({
      message: "Ticket assigned successfully",
      ticket,
    });
  } catch (error) {
    return res.status(500).json({
      status: "error",
      message: error.message,
    });
  }
});

/**
 * @swagger
 * /t/tickets/{agentId}/tickets:
 *   get:
 *     summary: Get tickets assigned to a specific agent with pagination
 *     tags:
 *       - Tickets
 *     parameters:
 *       - in: path
 *         name: agentId
 *         required: true
 *         schema:
 *           type: string
 *         description: ID of the Agent
 *       - in: query
 *         name: page
 *         schema:
 *           type: integer
 *           default: 1
 *         description: Page number
 *       - in: query
 *         name: limit
 *         schema:
 *           type: integer
 *           default: 10
 *         description: Number of items per page
 *     responses:
 *       200:
 *         description: Paginated list of assigned tickets
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 status:
 *                   type: string
 *                 currentPage:
 *                   type: integer
 *                 itemsPerPage:
 *                   type: integer
 *                 totalItems:
 *                   type: integer
 *                 totalPages:
 *                   type: integer
 *                 tickets:
 *                   type: array
 *                   items:
 *                     type: object
 *       404:
 *         description: Agent not found
 *       500:
 *         description: Server error
 */
router.get("/:agentId/tickets", Auth, async (req, res) => {
  try {
    const { agentId } = req.params;
    const page = parseInt(req.query.page) || 1;
    const limit = parseInt(req.query.limit) || 10;
    const offset = (page - 1) * limit;

    const agent = await User.findByPk(agentId);
    if (!agent) {
      return res.status(404).json({
        status: "fail",
        message: "Agent not found",
      });
    }

    const { count, rows: tickets } = await TicketModel.findAndCountAll({
      where: { assignedTo: agentId },
      limit,
      offset,
      order: [["createdAt", "DESC"]], // Optional: add sorting
    });

    const totalPages = Math.ceil(count / limit);

    const responseBody = {
      status: "success",
      currentPage: page,
      itemsPerPage: limit,
      totalItems: count,
      totalPages,
      tickets,
    };

    res.status(200).json(responseBody);
  } catch (error) {
    console.error("Error fetching tickets:", error);
    res.status(500).json({
      status: "error",
      message: "Internal server error.",
    });
  }
});

export default router;
