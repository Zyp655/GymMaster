import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import dayjs from 'dayjs';
import 'dayjs/locale/vi';
import relativeTime from 'dayjs/plugin/relativeTime';
import customParseFormat from 'dayjs/plugin/customParseFormat';
import App from './App';
import { setupSession } from './store/session';
import './assets/css/custom.css';

dayjs.extend(relativeTime);
dayjs.extend(customParseFormat);
dayjs.locale('vi');

setupSession();

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
