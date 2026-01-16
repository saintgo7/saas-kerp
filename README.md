# SaaS K-ERP

A modern, cloud-based Enterprise Resource Planning (ERP) system built with Next.js, TypeScript, and Tailwind CSS.

## Overview

SaaS K-ERP is a comprehensive ERP solution designed for small to medium-sized businesses. It provides essential modules for managing core business operations:

- **Inventory Management** - Track stock levels and warehouse operations
- **Sales & Orders** - Manage sales orders and customer relationships
- **Purchasing** - Handle purchase orders and supplier management
- **Financial Management** - Accounting, invoicing, and financial reporting
- **HR Management** - Employee records and payroll processing
- **Reporting & Analytics** - Business intelligence and data insights

## Tech Stack

- **Framework**: Next.js 15 (React 19)
- **Language**: TypeScript
- **Styling**: Tailwind CSS
- **Package Manager**: npm

## Getting Started

### Prerequisites

- Node.js 18+ 
- npm 9+

### Installation

1. Clone the repository:
```bash
git clone https://github.com/saintgo7/saas-kerp.git
cd saas-kerp
```

2. Install dependencies:
```bash
npm install
```

3. Create environment variables file:
```bash
cp .env.example .env.local
```

4. Run the development server:
```bash
npm run dev
```

5. Open [http://localhost:3000](http://localhost:3000) in your browser to see the application.

## Available Scripts

- `npm run dev` - Starts the development server
- `npm run build` - Creates an optimized production build
- `npm start` - Runs the production server
- `npm run lint` - Runs ESLint to check code quality

## Project Structure

```
saas-kerp/
├── app/                    # Next.js app directory
│   ├── dashboard/         # Dashboard pages
│   ├── api/              # API routes
│   ├── globals.css       # Global styles
│   ├── layout.tsx        # Root layout
│   └── page.tsx          # Home page
├── public/               # Static assets
├── .gitignore           # Git ignore rules
├── next.config.ts       # Next.js configuration
├── package.json         # Project dependencies
├── tailwind.config.ts   # Tailwind CSS configuration
└── tsconfig.json        # TypeScript configuration
```

## Features

### Current Features
- ✅ Landing page with module overview
- ✅ Dashboard with KPI cards
- ✅ Responsive design (mobile, tablet, desktop)
- ✅ Dark mode support

### Roadmap
- 🚧 User authentication & authorization
- 🚧 Database integration
- 🚧 Inventory management module
- 🚧 Sales & order processing
- 🚧 Purchase order management
- 🚧 Financial reporting
- 🚧 HR management system
- 🚧 Analytics dashboard

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is private and proprietary.

## Contact

For questions or support, please contact the repository owner.