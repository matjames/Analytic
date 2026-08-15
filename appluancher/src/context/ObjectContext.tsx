import React, { createContext, useContext, useState, useCallback } from 'react';

export interface EnterpriseObjectTarget {
  type: string;
  id: string;
  title?: string;
  initialData?: any;
}

interface ObjectContextValue {
  activeObject: EnterpriseObjectTarget | null;
  openObjectContext: (type: string, id: string, initialData?: any) => void;
  closeObjectContext: () => void;
}

const ObjectContext = createContext<ObjectContextValue | undefined>(undefined);

export const ObjectContextProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [activeObject, setActiveObject] = useState<EnterpriseObjectTarget | null>(null);

  const openObjectContext = useCallback((type: string, id: string, initialData?: any) => {
    if (!type || !id) return;
    setActiveObject({
      type: type.toLowerCase(),
      id,
      title: initialData?.title || initialData?.name || `${type.toUpperCase()} #${id}`,
      initialData,
    });
  }, []);

  const closeObjectContext = useCallback(() => {
    setActiveObject(null);
  }, []);

  return (
    <ObjectContext.Provider value={{ activeObject, openObjectContext, closeObjectContext }}>
      {children}
    </ObjectContext.Provider>
  );
};

export const useObjectContext = (): ObjectContextValue => {
  const ctx = useContext(ObjectContext);
  if (!ctx) {
    throw new Error('useObjectContext must be used within an ObjectContextProvider');
  }
  return ctx;
};
