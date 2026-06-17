# language: pt
Funcionalidade: Distribuição de atendimentos da FlowPay
  Como gestor da central de relacionamento
  Quero que solicitações sejam distribuídas ao time correto respeitando a capacidade
  Para garantir atendimento ordenado e sem sobrecarga

  Contexto:
    Dado o time "Cartões" responsável por "Problemas com cartão"
    E o time "Empréstimos" responsável por "Contratação de empréstimo"
    E o time "Outros Assuntos" para os demais assuntos

  Cenário: Atribuir solicitação a atendente livre do time correto
    Dado um atendente "Ana" no time "Cartões" sem atendimentos
    Quando chega uma solicitação de "Problemas com cartão"
    Então a solicitação é atribuída a "Ana"
    E "Ana" passa a ter 1 atendimento ativo

  Cenário: Respeitar o limite de 3 atendimentos simultâneos
    Dado um atendente "Bruno" no time "Cartões" com 3 atendimentos ativos
    Quando chega uma solicitação de "Problemas com cartão"
    Então a solicitação não é atribuída a "Bruno"

  Cenário: Enfileirar quando todos os atendentes do time estão lotados
    Dado que todos os atendentes do time "Cartões" estão com 3 atendimentos ativos
    Quando chega uma solicitação de "Problemas com cartão"
    Então a solicitação é enfileirada no time "Cartões"

  Cenário: Distribuir da fila assim que um atendente fica livre
    Dado uma solicitação aguardando na fila do time "Cartões"
    E um atendente "Ana" do time "Cartões" com 3 atendimentos ativos
    Quando "Ana" finaliza um atendimento
    Então a próxima solicitação da fila é atribuída a "Ana"

  Cenário: Rotear assunto desconhecido para o time Outros Assuntos
    Dado um atendente "Carla" no time "Outros Assuntos" sem atendimentos
    Quando chega uma solicitação de "Atualização cadastral"
    Então a solicitação é atribuída a "Carla"

  Cenário: Balancear distribuição ao atendente com menor carga
    Dado um atendente "Ana" no time "Cartões" com 2 atendimentos ativos
    E um atendente "Bruno" no time "Cartões" sem atendimentos
    Quando chega uma solicitação de "Problemas com cartão"
    Então a solicitação é atribuída a "Bruno"
    E "Ana" continua com 2 atendimentos ativos

  Cenário: Balancear entre atendentes com carga igual
    Dado um atendente "Ana" no time "Cartões" com 1 atendimento ativo
    E um atendente "Bruno" no time "Cartões" com 1 atendimento ativo
    E um atendente "Carlos" no time "Cartões" sem atendimentos
    Quando chega uma solicitação de "Problemas com cartão"
    Então a solicitação é atribuída a "Carlos"

  Cenário: Adicionar atendente ao time em tempo de execução
    Dado o time "Cartões" com apenas um atendente "Ana" com 3 atendimentos ativos
    Quando um novo atendente "Bruno" é adicionado ao time "Cartões"
    E chega uma solicitação de "Problemas com cartão"
    Então a solicitação é atribuída a "Bruno"

  Cenário: Novo atendente participa do balanceamento imediatamente
    Dado um atendente "Ana" no time "Cartões" com 2 atendimentos ativos
    Quando um novo atendente "Bruno" é adicionado ao time "Cartões"
    E chega uma solicitação de "Problemas com cartão"
    Então a solicitação é atribuída a "Bruno" pois tem menor carga

  Cenário: Expor solicitações ativas por atendente no snapshot
    Dado um atendente "Ana" no time "Cartões" sem atendimentos
    Quando chega uma solicitação de "Problemas com cartão"
    Então o snapshot do time "Cartões" lista a solicitação ativa de "Ana"

  Cenário: Registrar duração do atendimento ao finalizar
    Dado um atendente "Ana" no time "Cartões" sem atendimentos
    E uma solicitação de "Problemas com cartão" atribuída a "Ana"
    Quando "Ana" finaliza o atendimento
    Então o evento "atendimento_finalizado" contém o campo "duracao_atendimento_seg" preenchido
    E o valor de "duracao_atendimento_seg" é maior ou igual a zero

  Cenário: Registrar tempo na fila ao puxar solicitação enfileirada
    Dado que todos os atendentes do time "Cartões" estão com 3 atendimentos ativos
    E uma solicitação de "Problemas com cartão" está aguardando na fila
    Quando um atendente finaliza um atendimento e puxa a solicitação da fila
    Então o evento "solicitacao_atribuida" contém o campo "tempo_na_fila_seg" preenchido
    E o evento "atendimento_finalizado" do atendimento anterior não contém "tempo_na_fila_seg"

  Cenário: Atendente pausado não recebe novas solicitações
    Dado um atendente "Ana" no time "Cartões" sem atendimentos
    E um atendente "Bruno" no time "Cartões" com 1 atendimento ativo
    Quando "Ana" é pausada
    E chega uma solicitação de "Problemas com cartão"
    Então a solicitação é atribuída a "Bruno"
    E "Ana" continua sem atendimentos ativos

  Cenário: Pausar atendente com atendimentos ativos é bloqueado
    Dado um atendente "Ana" no time "Cartões" com 1 atendimento ativo
    Quando se tenta pausar "Ana"
    Então a operação falha com erro "atendente possui atendimentos ativos"
    E "Ana" permanece não pausada

  Cenário: Atendente pausado não interfere no balanceamento de carga
    Dado um atendente "Ana" no time "Cartões" sem atendimentos
    Quando "Ana" é pausada
    E chega uma solicitação de "Problemas com cartão"
    Então a solicitação é enfileirada no time "Cartões"

  Cenário: Retomar atendente pausado puxa solicitações da fila
    Dado um atendente "Ana" no time "Cartões" sem atendimentos
    E "Ana" é pausada
    E uma solicitação de "Problemas com cartão" foi enfileirada
    Quando "Ana" é retomada
    Então a solicitação da fila é atribuída a "Ana"

  Cenário: Remover atendente sem atendimentos ativos
    Dado um atendente "Ana" no time "Cartões" com 2 atendimentos ativos
    E um atendente "Bruno" no time "Cartões" sem atendimentos
    Quando "Bruno" é removido do time "Cartões"
    Então o time "Cartões" conta com apenas "Ana" como atendente

  Cenário: Remover atendente com atendimentos ativos é bloqueado
    Dado um atendente "Ana" no time "Cartões" com 1 atendimento ativo
    Quando se tenta remover "Ana"
    Então a operação falha com erro "atendente possui atendimentos ativos"

  Cenário: Adicionar atendente puxa da fila imediatamente
    Dado o time "Cartões" com apenas um atendente "Ana" com 3 atendimentos ativos
    E uma solicitação de "Problemas com cartão" está aguardando na fila
    Quando um novo atendente "Bruno" é adicionado ao time "Cartões"
    Então a solicitação da fila é atribuída a "Bruno" sem aguardar nova chegada
