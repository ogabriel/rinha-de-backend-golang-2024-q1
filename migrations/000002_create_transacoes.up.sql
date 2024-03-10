CREATE TABLE transacoes (
    id serial PRIMARY KEY,
    cliente_id integer REFERENCES clientes (id),
    valor integer,
    tipo char(1),
    descricao varchar(10),
    realizada_em varchar(27)
);

CREATE INDEX ON transacoes (cliente_id);
