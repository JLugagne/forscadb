import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { HashRouter } from 'react-router-dom'
import { Toaster } from 'react-hot-toast'
import App from './App'
import './index.css'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <HashRouter>
      <App />
      <Toaster
        position="bottom-right"
        toastOptions={{
          style: {
            background: '#12121e',
            color: '#e8e6f0',
            border: '1px solid #2a2a42',
            borderRadius: '12px',
            fontFamily: "'DM Sans', sans-serif",
            fontSize: '13px',
          },
        }}
      />
    </HashRouter>
  </StrictMode>,
)
