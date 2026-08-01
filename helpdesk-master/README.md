# StatGate Operations Service

A self-contained operations platform for handling service requests, case triage, knowledge sharing, and admin oversight in a StatGate-style workflow.

## 🚀 Technologies Used

### Backend
- **Node.js** - JavaScript runtime environment
- **Express.js** - Web application framework
- **PostgreSQL** - Relational database
- **Sequelize** - Object-Relational Mapping (ORM)
- **JWT** - JSON Web Token authentication
- **Bcrypt** - Password hashing
- **Multer** - File upload handling
- **Nodemailer** - Email functionality
- **Swagger** - API documentation
- **Morgan** - HTTP request logger
- **CORS** - Cross-Origin Resource Sharing

### Frontend
- **React.js** - JavaScript library for building user interfaces
- **Bootstrap** - CSS framework
- **React Router** - Client-side routing
- **Axios** - HTTP client
- **React Icons** - Icon library
- **Recharts** - Chart library
- **React Toastify** - Toast notifications
- **XLSX** - Excel file handling

## 📋 Prerequisites

Before you begin, ensure you have the following installed on your system:

- **Node.js** (v16 or higher)
- **npm** or **yarn** package manager
- **PostgreSQL** (v12 or higher)
- **Git** for version control

## 🛠️ Installation & Setup

### 1. Clone the Repository

```bash
git clone <repository-url>
cd helpdesk
```

### 2. Backend Setup

#### Navigate to Backend Directory
```bash
cd backend
```

#### Install Dependencies
```bash
npm install
# or
yarn install
```

#### Environment Configuration
Create a `.env` file in the backend directory with the following variables:

```env
# Server Configuration
PORT=5000
NODE_ENV=development

# Database Configuration
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=statgate_db
POSTGRES_USER=your_username
POSTGRES_PASSWORD=your_password

# JWT Configuration
JWT_SECRET=your_jwt_secret_key
JWT_EXPIRE=24h

# Email Configuration (if using email features)
EMAIL_HOST=smtp.gmail.com
EMAIL_PORT=587
EMAIL_USER=your_email@gmail.com
EMAIL_PASS=your_email_password
```

#### Database Setup
1. Create a PostgreSQL database:
```sql
CREATE DATABASE statgate_db;
```

2. The application will automatically create tables using Sequelize sync when you start the server.

#### Start the Backend Server
```bash
# Development mode with nodemon
npm run server

# Or start normally
npm start
```

The backend server will start on `http://localhost:5000`

### 3. Frontend Setup

#### Navigate to Frontend Directory
```bash
cd ../frontend
```

#### Install Dependencies
```bash
npm install
# or
yarn install
```

#### Environment Configuration
Create a `.env` file in the frontend directory:

```env
REACT_APP_API_URL=http://localhost:5000/api
REACT_APP_BASE_URL=http://localhost:3000
```

#### Start the Frontend Development Server
```bash
npm start
# or
yarn start
```

The frontend application will open in your browser at `http://localhost:3000`

## 🗄️ Database Schema

The system includes the following main models:

- **Users** - User management and authentication
- **Tickets** - Support ticket management
- **Comments** - Ticket comments and updates
- **Agents** - Support agent management
- **Knowledge Base** - Articles and documentation
- **Videos** - Video content management

## 📚 API Endpoints

The backend provides the following API endpoints:

- `POST /api/users` - User registration and authentication
- `GET /api/t/tickets` - Ticket management
- `POST /api/t/comments` - Comment management
- `GET /api/t/agents` - Agent management
- `GET /api/videos` - Video content
- `GET /api/knowledge-base` - Knowledge base articles

API documentation is available at `http://localhost:5000/api-docs` when the backend is running.

## 🏗️ Project Structure

```
helpdesk/
├── backend/
│   ├── config/
│   │   ├── db.js          # Database configuration
│   │   └── email.js       # Email configuration
│   ├── controllers/       # Business logic
│   ├── middleware/        # Authentication middleware
│   ├── models/           # Database models
│   ├── routes/           # API routes
│   ├── uploads/          # File uploads
│   ├── index.js          # Main server file
│   └── package.json
├── frontend/
│   ├── public/           # Static files
│   ├── src/
│   │   ├── admin/        # Admin panel components
│   │   ├── components/   # Reusable components
│   │   ├── pages/        # Page components
│   │   ├── helpers/      # Utility functions
│   │   └── App.js        # Main application component
│   └── package.json
└── README
```

## 🚀 Available Scripts

### Backend Scripts
```bash
npm run server      # Start development server with nodemon
npm start          # Start production server
```

### Frontend Scripts
```bash
npm start          # Start development server
npm run build      # Build for production
npm test           # Run tests
npm run eject      # Eject from Create React App
```

## 🔧 Development Workflow

1. **Start Backend**: Run `npm run server` in the backend directory
2. **Start Frontend**: Run `npm start` in the frontend directory
3. **Database**: Ensure PostgreSQL is running and accessible
4. **Environment**: Verify all environment variables are set correctly

## 🐛 Troubleshooting

### Common Issues

1. **Database Connection Error**
   - Verify PostgreSQL is running
   - Check database credentials in `.env` file
   - Ensure database exists

2. **Port Already in Use**
   - Change PORT in `.env` file
   - Kill existing processes using the port

3. **CORS Issues**
   - Verify backend CORS configuration
   - Check frontend API URL configuration

4. **Module Not Found Errors**
   - Run `npm install` in both directories
   - Clear node_modules and reinstall if necessary

## 📝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License.

## 👥 Support

For support and questions, please contact the development team or create an issue in the repository.

---

**Note**: This system is designed as a self-contained StatGate operations platform with features for case intake, knowledge management, and operational oversight. Ensure all security best practices are followed when deploying to production environments.