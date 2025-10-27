import { useState } from 'react';

interface LoginModalProps {
  isOpen: boolean;
  onClose: () => void;
  onLogin: (credentials: LoginCredentials) => void;
}

interface LoginCredentials {
  partitionId: string;
  username: string;
  password: string;
  remember: boolean;
}

const LoginModal = ({ isOpen, onClose, onLogin }: LoginModalProps) => {
  const [credentials, setCredentials] = useState<LoginCredentials>({
    partitionId: '',
    username: '',
    password: '',
    remember: false
  });
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    
    if (!credentials.partitionId || !credentials.username || !credentials.password) {
      setError('Todos los campos son obligatorios');
      return;
    }

    setIsLoading(true);
    try {
      await onLogin(credentials);
      onClose();
      setCredentials({
        partitionId: '',
        username: '',
        password: '',
        remember: false
      });
    } catch (err) {
      setError('Error al iniciar sesión. Verifica tus credenciales.');
    } finally {
      setIsLoading(false);
    }
  };

  const handleChange = (field: keyof LoginCredentials, value: string | boolean) => {
    setCredentials(prev => ({
      ...prev,
      [field]: value
    }));
    setError('');
  };

  if (!isOpen) return null;

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>Iniciar Sesión</h2>
          <button className="close-btn" onClick={onClose}>×</button>
        </div>
        
        <form onSubmit={handleSubmit} className="login-form">
          {error && <div className="error-message">{error}</div>}
          
          <div className="form-group">
            <label htmlFor="partitionId">ID Partición</label>
            <input
              type="text"
              id="partitionId"
              value={credentials.partitionId}
              onChange={(e) => handleChange('partitionId', e.target.value)}
              placeholder="Ingresa el ID de la partición"
              disabled={isLoading}
            />
          </div>

          <div className="form-group">
            <label htmlFor="username">Usuario</label>
            <input
              type="text"
              id="username"
              value={credentials.username}
              onChange={(e) => handleChange('username', e.target.value)}
              placeholder="Ingresa tu usuario"
              disabled={isLoading}
            />
          </div>

          <div className="form-group">
            <label htmlFor="password">Contraseña</label>
            <input
              type="password"
              id="password"
              value={credentials.password}
              onChange={(e) => handleChange('password', e.target.value)}
              placeholder="Ingresa tu contraseña"
              disabled={isLoading}
            />
          </div>

          <div className="form-group checkbox-group">
            <label className="checkbox-label">
              <input
                type="checkbox"
                checked={credentials.remember}
                onChange={(e) => handleChange('remember', e.target.checked)}
                disabled={isLoading}
              />
              <span className="checkmark"></span>
              Recordar sesión
            </label>
          </div>

          <div className="form-actions">
            <button 
              type="button" 
              onClick={onClose} 
              className="btn btn-secondary"
              disabled={isLoading}
            >
              Cancelar
            </button>
            <button 
              type="submit" 
              className="btn btn-primary"
              disabled={isLoading}
            >
              {isLoading ? 'Iniciando...' : 'Iniciar Sesión'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default LoginModal;