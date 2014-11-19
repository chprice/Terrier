from sqlalchemy import Column, Integer, String, DateTime

class Packet(Base):
    __tablename__ = 'packets'

    id = Column(Integer, primary_key=True)
    port = Column(Integer)
    ip = Column(String(15))
    ttl = Column(Integer)
    time = Column(DateTime)
